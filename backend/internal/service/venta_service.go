package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/worker"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type VentaService interface {
	RegistrarVenta(ctx context.Context, usuarioID uuid.UUID, req dto.RegistrarVentaRequest) (*dto.VentaResponse, error)
	AnularVenta(ctx context.Context, id uuid.UUID, motivo string) error
	SyncBatch(ctx context.Context, usuarioID uuid.UUID, req dto.SyncBatchRequest) (*dto.SyncBatchResponse, error)
	ListVentas(ctx context.Context, filter dto.VentaFilter) (*dto.VentaListResponse, error)
	SetClienteService(cs ClienteService)
}

type ventaService struct {
	repo            repository.VentaRepository
	inventario      InventarioService
	caja            CajaService
	cajaRepo        repository.CajaRepository
	productoRepo    repository.ProductoRepository
	comprobanteRepo repository.ComprobanteRepository
	configFiscalRepo repository.ConfiguracionFiscalRepository
	dispatcher      *worker.Dispatcher
	clienteSvc      ClienteService
}

func NewVentaService(
	repo repository.VentaRepository,
	inventario InventarioService,
	caja CajaService,
	cajaRepo repository.CajaRepository,
	productoRepo repository.ProductoRepository,
	dispatcher *worker.Dispatcher,
	comprobanteRepo repository.ComprobanteRepository,
	configFiscalRepo repository.ConfiguracionFiscalRepository,
) VentaService {
	return &ventaService{
		repo:            repo,
		inventario:      inventario,
		caja:            caja,
		cajaRepo:        cajaRepo,
		productoRepo:    productoRepo,
		comprobanteRepo: comprobanteRepo,
		configFiscalRepo: configFiscalRepo,
		dispatcher:      dispatcher,
	}
}

// SetClienteService injects the ClienteService after construction to avoid
// circular dependency in the composition root (main.go). When nil, fiado
// payment methods are rejected.
func (s *ventaService) SetClienteService(cs ClienteService) {
	s.clienteSvc = cs
}

// runTx executes fn inside a GORM transaction when db is available,
// or calls fn(nil) directly when db is nil (unit test mode).
func runTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if db == nil {
		return fn(nil)
	}
	return db.WithContext(ctx).Transaction(fn)
}

// ── RegistrarVenta ────────────────────────────────────────────────────────────
// Full ACID transaction per arquitectura.md §7.1:
//   1. Validate sesion de caja is open
//   2. For each item: fetch product price, calc subtotal, check stock
//   3. Validate total pagos >= total venta
//   4. BEGIN TX: nextval ticket, create venta+items+pagos, descontar stock, crear movimientos de caja
//   5. COMMIT
//   6. (async) dispatch facturacion job if needed

func (s *ventaService) RegistrarVenta(ctx context.Context, usuarioID uuid.UUID, req dto.RegistrarVentaRequest) (*dto.VentaResponse, error) {
	return s.registrarVentaInternal(ctx, usuarioID, req, false)
}

// registrarVentaInternal is the shared implementation for both online and offline sales.
// When fromSync=true (SyncBatch), stock conflicts are auto-compensated within threshold.
// When fromSync=false (online POS), insufficient stock is rejected with an error.
func (s *ventaService) registrarVentaInternal(ctx context.Context, usuarioID uuid.UUID, req dto.RegistrarVentaRequest, fromSync bool) (*dto.VentaResponse, error) {
	sesionID, err := uuid.Parse(req.SesionCajaID)
	if err != nil {
		return nil, fmt.Errorf("sesion_caja_id inválido: %w", err)
	}

	// 1. Validate open session
	if err := s.caja.FindSesionAbierta(ctx, sesionID); err != nil {
		return nil, err
	}

	// 1b. Inherit sucursal_id from the caja session
	var sucursalID *uuid.UUID
	if sesionCaja, sesErr := s.cajaRepo.FindSesionByID(ctx, sesionID); sesErr == nil && sesionCaja != nil {
		sucursalID = sesionCaja.SucursalID
	}

	// 2. Deduplicate offline sale
	if req.OfflineID != nil {
		if existing, err := s.repo.FindByOfflineID(ctx, *req.OfflineID); err == nil {
			return ventaToResponse(existing), nil
		}
	}

	// 3. Resolve products and calculate totals (pre-flight, outside TX)
	type resolvedItem struct {
		productoID   uuid.UUID
		nombre       string
		precio       decimal.Decimal
		cantidad     int
		descuento    decimal.Decimal
		subtotal     decimal.Decimal
		peso         *decimal.Decimal
		unidadMedida string
	}

	var resolved []resolvedItem
	subtotal := decimal.Zero
	descuentoTotal := decimal.Zero
	conflictoStock := false

	for _, item := range req.Items {
		pid, err := uuid.Parse(item.ProductoID)
		if err != nil {
			return nil, fmt.Errorf("producto_id inválido: %w", err)
		}
		p, err := s.productoRepo.FindByID(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("producto %s no encontrado", item.ProductoID)
		}
		if !p.Activo {
			return nil, fmt.Errorf("producto %s está inactivo y no puede venderse", p.Nombre)
		}

		// Weight-based validation: kg/gramo products require peso > 0
		isWeightBased := p.UnidadMedida == "kg" || p.UnidadMedida == "gramo"
		if isWeightBased {
			if item.Peso == nil || !item.Peso.IsPositive() {
				return nil, fmt.Errorf("producto %s se vende por %s: debe indicar el peso", p.Nombre, p.UnidadMedida)
			}
		}

		if !isWeightBased && p.StockActual < item.Cantidad {
			if !fromSync {
				// Online sales: check if auto-desarme can supply the deficit before rejecting.
				canDesarme := false
				if vinculo, vErr := s.productoRepo.FindVinculoByHijoID(ctx, pid); vErr == nil && vinculo.DesarmeAuto {
					deficit := item.Cantidad - p.StockActual
					padresNecesarios := (deficit + vinculo.UnidadesPorPadre - 1) / vinculo.UnidadesPorPadre
					if padre, pErr := s.productoRepo.FindByID(ctx, vinculo.ProductoPadreID); pErr == nil {
						canDesarme = padre.StockActual >= padresNecesarios
					}
				}
				if !canDesarme {
					return nil, fmt.Errorf("stock insuficiente para %s: disponible %d, solicitado %d", p.Nombre, p.StockActual, item.Cantidad)
				}
			}
			conflictoStock = true
		}

		// Calculate line total: weight-based uses precio * peso, unit-based uses precio * cantidad
		var lineTotal decimal.Decimal
		if isWeightBased && item.Peso != nil {
			lineTotal = p.PrecioVenta.Mul(*item.Peso)
		} else {
			lineTotal = p.PrecioVenta.Mul(decimal.NewFromInt(int64(item.Cantidad)))
		}

		// Descuento cap: no puede superar el 50% del valor de la línea.
		// Prevents negative subtotals and guards against client-side manipulation.
		maxDescuento := lineTotal.Mul(decimal.NewFromFloat(0.50))
		if item.Descuento.GreaterThan(maxDescuento) {
			return nil, fmt.Errorf("descuento para %s excede el máximo permitido (50%% del precio de línea)", p.Nombre)
		}
		lineSubtotal := lineTotal.Sub(item.Descuento)
		subtotal = subtotal.Add(lineSubtotal)
		descuentoTotal = descuentoTotal.Add(item.Descuento)
		resolved = append(resolved, resolvedItem{
			productoID:   pid,
			nombre:       p.Nombre,
			precio:       p.PrecioVenta,
			cantidad:     item.Cantidad,
			descuento:    item.Descuento,
			subtotal:     lineSubtotal,
			peso:         item.Peso,
			unidadMedida: p.UnidadMedida,
		})
	}

	total := subtotal

	// 4. Validate payment sufficiency
	totalPagos := decimal.Zero
	for _, pago := range req.Pagos {
		totalPagos = totalPagos.Add(pago.Monto)
	}
	if totalPagos.LessThan(total) {
		return nil, errors.New("El monto total de pagos es insuficiente")
	}
	vuelto := totalPagos.Sub(total)

	// 4b. Validate fiado payment — requires cliente_id and ClienteService
	var clienteID *uuid.UUID
	var montoFiado decimal.Decimal
	for _, pago := range req.Pagos {
		if pago.Metodo == "fiado" {
			montoFiado = montoFiado.Add(pago.Monto)
		}
	}
	if montoFiado.IsPositive() {
		if req.ClienteID == nil || *req.ClienteID == "" {
			return nil, errors.New("cliente_id es requerido para ventas con método de pago fiado")
		}
		if s.clienteSvc == nil {
			return nil, errors.New("el sistema de fiado no está configurado")
		}
		cid, err := uuid.Parse(*req.ClienteID)
		if err != nil {
			return nil, fmt.Errorf("cliente_id inválido: %w", err)
		}
		clienteID = &cid
	}

	// Resolve tipo_comprobante — auto-determine from fiscal config if not specified
	tipoComp := "ticket_interno"
	if req.TipoComprobante != nil && *req.TipoComprobante != "" {
		tipoComp = *req.TipoComprobante
	} else {
		// Auto-determine from fiscal configuration
		if s.configFiscalRepo != nil {
			if cfg, err := s.configFiscalRepo.Get(ctx); err == nil && cfg != nil && cfg.CUITEmsior != "" {
				// Configuración fiscal existe — determinar tipo según condición fiscal
				switch cfg.CondicionFiscal {
				case "Responsable Inscripto":
					// RI → Factura B por defecto (o A si receptor tiene CUIT)
					if req.TipoDocReceptor != nil && *req.TipoDocReceptor == 80 {
						tipoComp = "factura_a"
					} else {
						tipoComp = "factura_b"
					}
				case "Monotributo", "Exento":
					// Monotributo/Exento → Factura C
					tipoComp = "factura_c"
				default:
					// Sin config o config inválida → ticket interno
					tipoComp = "ticket_interno"
				}
				log.Info().Str("condicion_fiscal", cfg.CondicionFiscal).Str("tipo_comprobante", tipoComp).Msg("Auto-determinando tipo de comprobante desde configuración fiscal")
			}
		}
	}

	// 5. ACID transaction with row-level stock lock
	var venta model.Venta
	txErr := runTx(ctx, s.repo.DB(), func(tx *gorm.DB) error {
		// Re-validate stock INSIDE the transaction with SELECT ... FOR UPDATE
		// to prevent race conditions between concurrent POS terminals.
		// Guard: skip when tx is nil (unit test mode without real DB).
		if !fromSync && tx != nil {
			for _, r := range resolved {
				var stockActual int
				row := tx.Raw("SELECT stock_actual FROM productos WHERE id = ? FOR UPDATE", r.productoID).Row()
				if row == nil {
					return fmt.Errorf("producto %s no encontrado en TX", r.nombre)
				}
				if err := row.Scan(&stockActual); err != nil {
					return fmt.Errorf("error leyendo stock de %s: %w", r.nombre, err)
				}
				if stockActual < r.cantidad {
					return fmt.Errorf("stock insuficiente para %s: disponible %d, solicitado %d", r.nombre, stockActual, r.cantidad)
				}
			}
		}

		ticketNum, err := s.repo.NextTicketNumber(ctx, tx)
		if err != nil {
			return err
		}

		// Build venta model
		venta = model.Venta{
			NumeroTicket:    ticketNum,
			SesionCajaID:    sesionID,
			UsuarioID:       usuarioID,
			SucursalID:      sucursalID,
			ClienteID:       clienteID,
			Subtotal:        subtotal,
			DescuentoTotal:  descuentoTotal,
			Total:           total,
			Estado:          "completada",
			TipoComprobante: tipoComp,
			OfflineID:       req.OfflineID,
			ConflictoStock:  conflictoStock,
		}

		// Build items
		for _, r := range resolved {
			venta.Items = append(venta.Items, model.VentaItem{
				ProductoID:     r.productoID,
				Cantidad:       r.cantidad,
				PrecioUnitario: r.precio,
				DescuentoItem:  r.descuento,
				Subtotal:       r.subtotal,
				Peso:           r.peso,
			})
		}

		// Build pagos
		for _, pago := range req.Pagos {
			venta.Pagos = append(venta.Pagos, model.VentaPago{
				Metodo: pago.Metodo,
				Monto:  pago.Monto,
			})
		}

		if err := s.repo.Create(ctx, tx, &venta); err != nil {
			return err
		}

		// Descontar stock — uses DescontarStockTx (handles auto-desarme from Fase 3)
		for _, r := range resolved {
			// Fetch current stock INSIDE tx for movement record
			prodBefore, err := s.productoRepo.FindByIDTx(tx, r.productoID)
			stockAntes := 0
			if err == nil && prodBefore != nil {
				stockAntes = prodBefore.StockActual
			}

			if err := s.inventario.DescontarStockTx(ctx, r.productoID, r.cantidad, tx); err != nil {
				return fmt.Errorf("error descontando stock de %s: %w", r.nombre, err)
			}

			// Record movimiento de stock
			ventaRef := venta.ID
			mov := &model.MovimientoStock{
				ProductoID:    r.productoID,
				Tipo:          "venta",
				Cantidad:      -r.cantidad,
				StockAnterior: stockAntes,
				StockNuevo:    stockAntes - r.cantidad,
				Motivo:        fmt.Sprintf("Venta #%d", ticketNum),
				ReferenciaID:  &ventaRef,
			}
			if err := s.inventario.RegistrarMovimientoTx(tx, mov); err != nil {
				return err
			}
		}

		// Create movimientos de caja (one per payment method)
		for _, pago := range req.Pagos {
			metodo := pago.Metodo
			mov := model.MovimientoCaja{
				SesionCajaID: sesionID,
				Tipo:         "venta",
				MetodoPago:   &metodo,
				Monto:        pago.Monto,
				Descripcion:  fmt.Sprintf("Venta #%d", ticketNum),
				ReferenciaID: &venta.ID,
			}
			if err := s.cajaRepo.CreateMovimientoTx(tx, &mov); err != nil {
				return err
			}
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	// 5b. Post-TX: charge fiado to customer's credit account
	if montoFiado.IsPositive() && clienteID != nil && s.clienteSvc != nil {
		if err := s.clienteSvc.CargarFiado(ctx, *clienteID, venta.ID, montoFiado); err != nil {
			log.Error().Err(err).
				Str("venta_id", venta.ID.String()).
				Str("cliente_id", clienteID.String()).
				Msg("CRITICO: venta registrada pero fallo al cargar fiado en cuenta corriente")
			// The sale is already committed. The fiado charge failure is logged for
			// manual reconciliation — we don't rollback the sale.
		}
	}

	// 6. Async facturacion job — error is handled: if Redis is down we create a
	// pending comprobante directly so that retry_cron picks it up on next cycle.
	if s.dispatcher != nil {
		fiscalPayload := worker.FacturacionJobPayload{
			VentaID:         venta.ID.String(),
			TipoComprobante: tipoComp,
			ClienteEmail:    req.ClienteEmail,
			TipoDocReceptor: req.TipoDocReceptor,
			NroDocReceptor:  req.NroDocReceptor,
			ReceptorNombre:  req.ReceptorNombre,
			ReceptorDomicilio: req.ReceptorDomicilio,
		}
		if err := s.dispatcher.EnqueueFacturacion(ctx, fiscalPayload); err != nil {
			log.Error().Err(err).Str("venta_id", venta.ID.String()).
				Msg("CRITICO: fallo al encolar facturacion — creando comprobante pendiente para retry")
			// Fallback: create the comprobante record directly in estado='pendiente'.
			// The retry_cron will process it on the next 30-second tick.
			if s.comprobanteRepo != nil {
				nextRetry := time.Now().Add(30 * time.Second)
				comp := &model.Comprobante{
					VentaID:              venta.ID,
					Tipo:                 tipoComp,
					MontoNeto:            venta.Total,
					MontoIVA:             decimal.Zero,
					MontoTotal:           venta.Total,
					Estado:               "pendiente",
					ReceptorTipoDocumento: req.TipoDocReceptor,
					ReceptorNumeroDocumento: req.NroDocReceptor,
					ReceptorCUIT:         req.NroDocReceptor,
					ReceptorNombre:       req.ReceptorNombre,
					ReceptorDomicilio:    req.ReceptorDomicilio,
					RetryCount:           0,
					NextRetryAt:          &nextRetry,
				}
				if err2 := s.comprobanteRepo.Create(ctx, comp); err2 != nil {
					log.Error().Err(err2).Str("venta_id", venta.ID.String()).
						Msg("CRITICO: no se pudo crear comprobante fallback — revisar manualmente")
				}
			}
		}
	}

	// Build response
	resp := ventaToResponse(&venta)
	resp.Vuelto = vuelto
	// Enrich items with product names from resolved slice
	for i, r := range resolved {
		resp.Items[i].Producto = r.nombre
	}
	return resp, nil
}

// ── AnularVenta ───────────────────────────────────────────────────────────────

func (s *ventaService) AnularVenta(ctx context.Context, id uuid.UUID, motivo string) error {
	venta, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New("venta no encontrada")
	}
	if venta.Estado == "anulada" {
		return errors.New("la venta ya está anulada")
	}

	txErr := runTx(ctx, s.repo.DB(), func(tx *gorm.DB) error {
		// H-06: Restore stock for each item. Read stock INSIDE the transaction
		// with FOR UPDATE to prevent phantom reads from concurrent operations.
		for _, item := range venta.Items {
			var stockAntes int
			// Guard: skip FOR UPDATE when tx is nil (unit test mode without real DB).
			if tx != nil {
				row := tx.Raw("SELECT stock_actual FROM productos WHERE id = ? FOR UPDATE", item.ProductoID).Row()
				if row != nil {
					_ = row.Scan(&stockAntes)
				}
			}

			if err := s.productoRepo.UpdateStockTx(tx, item.ProductoID, item.Cantidad); err != nil {
				return err
			}

			ventaRef := venta.ID
			movStock := &model.MovimientoStock{
				ProductoID:    item.ProductoID,
				Tipo:          "restore_anulacion",
				Cantidad:      item.Cantidad,
				StockAnterior: stockAntes,
				StockNuevo:    stockAntes + item.Cantidad,
				Motivo:        fmt.Sprintf("Anulación venta #%d — %s", venta.NumeroTicket, motivo),
				ReferenciaID:  &ventaRef,
			}
			if err := s.inventario.RegistrarMovimientoTx(tx, movStock); err != nil {
				return err
			}
		}

		// Create inverse movimientos de caja
		for _, pago := range venta.Pagos {
			metodo := pago.Metodo
			monto := pago.Monto.Neg()
			mov := model.MovimientoCaja{
				SesionCajaID: venta.SesionCajaID,
				Tipo:         "anulacion",
				MetodoPago:   &metodo,
				Monto:        monto,
				Descripcion:  fmt.Sprintf("Anulación venta #%d — %s", venta.NumeroTicket, motivo),
				ReferenciaID: &venta.ID,
			}
			if err := s.cajaRepo.CreateMovimientoTx(tx, &mov); err != nil {
				return err
			}
		}

		return s.repo.UpdateEstadoTx(tx, id, "anulada")
	})
	return txErr
}

// ── SyncBatch ─────────────────────────────────────────────────────────────────
// Processes a batch of offline sales. Idempotent: uses offline_id deduplication.
//
// Offline-first principle: sales made at the physical POS MUST be recorded
// regardless of stock levels. If stock goes negative, the sale is flagged
// (ConflictoStock=true) for supervisor review, but never rejected.
// Rejecting an offline sale would mean losing a financial record of a
// transaction that already happened in the real world.
//
// F1-8: Multi-tenant sync-batch
// - tenant_id is injected from JWT context, never from the request body
// - Idempotency via batch-check of existing offline_ids, then INSERT ON CONFLICT DO NOTHING
// - Returns structured SyncBatchResponse with synced/duplicated/failed classification

func (s *ventaService) SyncBatch(ctx context.Context, usuarioID uuid.UUID, req dto.SyncBatchRequest) (*dto.SyncBatchResponse, error) {
	resp := &dto.SyncBatchResponse{
		SyncedIDs:     make([]string, 0, len(req.Ventas)),
		DuplicatedIDs: make([]string, 0),
		FailedIDs:     make([]string, 0),
	}

	// Empty batch — return 200 with empty arrays
	if len(req.Ventas) == 0 {
		return resp, nil
	}

	// 1. Collect all offline_ids from the batch
	offlineIDs := make([]string, 0, len(req.Ventas))
	ventaByOfflineID := make(map[string]*dto.RegistrarVentaRequest, len(req.Ventas))
	for i := range req.Ventas {
		v := &req.Ventas[i]
		oid := ""
		if v.OfflineID != nil {
			oid = *v.OfflineID
		}
		if oid != "" {
			offlineIDs = append(offlineIDs, oid)
			ventaByOfflineID[oid] = v
		}
	}

	// 2. Batch-query existing offline_ids to identify duplicates upfront
	existingIDs, err := s.repo.FindExistingOfflineIDs(ctx, offlineIDs)
	if err != nil {
		log.Error().Err(err).Msg("sync-batch: error checking existing offline_ids")
		return nil, fmt.Errorf("error verificando duplicados: %w", err)
	}

	existingSet := make(map[string]bool, len(existingIDs))
	for _, eid := range existingIDs {
		existingSet[eid] = true
	}

	// 3. Separate duplicates from new ventas
	for _, oid := range offlineIDs {
		if existingSet[oid] {
			resp.DuplicatedIDs = append(resp.DuplicatedIDs, oid)
			resp.TotalDuplicated++
		}
	}

	// 4. Process only new ventas — each one through registrarVentaInternal within
	//    the same transactional context (per-venta TX for stock + caja integrity).
	for i, ventaReq := range req.Ventas {
		offlineID := ""
		if ventaReq.OfflineID != nil {
			offlineID = *ventaReq.OfflineID
		}

		// Skip already-synced ventas (duplicates)
		if existingSet[offlineID] {
			continue
		}

		_, regErr := s.registrarVentaInternal(ctx, usuarioID, ventaReq, true)
		if regErr != nil {
			log.Warn().
				Int("index", i).
				Str("offline_id", offlineID).
				Err(regErr).
				Msg("sync-batch: venta rechazada")
			if offlineID != "" {
				resp.FailedIDs = append(resp.FailedIDs, offlineID)
			}
			continue
		}
		resp.TotalProcessed++
		if offlineID != "" {
			resp.SyncedIDs = append(resp.SyncedIDs, offlineID)
		}
	}

	return resp, nil
}

// ListVentas returns a paginated list of sales, filtered by date and estado.
// Default filter: today's completed sales.
func (s *ventaService) ListVentas(ctx context.Context, filter dto.VentaFilter) (*dto.VentaListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 50
	}
	if filter.Estado == "" {
		filter.Estado = "completada"
	}
	ventas, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	items := make([]dto.VentaListItem, 0, len(ventas))
	for _, v := range ventas {
		items = append(items, *ventaToListItem(&v))
	}
	return &dto.VentaListResponse{
		Data:  items,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

func ventaToListItem(v *model.Venta) *dto.VentaListItem {
	items := make([]dto.ItemVentaResponse, 0, len(v.Items))
	for _, item := range v.Items {
		nombre := ""
		unidadMedida := ""
		if item.Producto != nil {
			nombre = item.Producto.Nombre
			if item.Producto.UnidadMedida != "unidad" {
				unidadMedida = item.Producto.UnidadMedida
			}
		}
		items = append(items, dto.ItemVentaResponse{
			Producto:       nombre,
			Cantidad:       item.Cantidad,
			PrecioUnitario: item.PrecioUnitario,
			Subtotal:       item.Subtotal,
			Peso:           item.Peso,
			UnidadMedida:   unidadMedida,
		})
	}
	pagos := make([]dto.PagoRequest, 0, len(v.Pagos))
	for _, p := range v.Pagos {
		pagos = append(pagos, dto.PagoRequest{Metodo: p.Metodo, Monto: p.Monto})
	}
	cajeroNombre := ""
	if v.Usuario != nil {
		cajeroNombre = v.Usuario.Nombre
	}
	return &dto.VentaListItem{
		ID:             v.ID.String(),
		NumeroTicket:   v.NumeroTicket,
		SesionCajaID:   v.SesionCajaID.String(),
		UsuarioID:      v.UsuarioID.String(),
		CajeroNombre:   cajeroNombre,
		Total:          v.Total,
		DescuentoTotal: v.DescuentoTotal,
		Subtotal:       v.Subtotal,
		Estado:         v.Estado,
		Items:          items,
		Pagos:          pagos,
		CreatedAt:      v.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func ventaToResponse(v *model.Venta) *dto.VentaResponse {
	items := make([]dto.ItemVentaResponse, 0, len(v.Items))
	for _, item := range v.Items {
		nombre := ""
		unidadMedida := ""
		if item.Producto != nil {
			nombre = item.Producto.Nombre
			if item.Producto.UnidadMedida != "unidad" {
				unidadMedida = item.Producto.UnidadMedida
			}
		}
		items = append(items, dto.ItemVentaResponse{
			Producto:       nombre,
			Cantidad:       item.Cantidad,
			PrecioUnitario: item.PrecioUnitario,
			Subtotal:       item.Subtotal,
			Peso:           item.Peso,
			UnidadMedida:   unidadMedida,
		})
	}
	pagos := make([]dto.PagoRequest, 0, len(v.Pagos))
	for _, p := range v.Pagos {
		pagos = append(pagos, dto.PagoRequest{Metodo: p.Metodo, Monto: p.Monto})
	}
	return &dto.VentaResponse{
		ID:             v.ID.String(),
		NumeroTicket:   v.NumeroTicket,
		Items:          items,
		Subtotal:       v.Subtotal,
		DescuentoTotal: v.DescuentoTotal,
		Total:          v.Total,
		Pagos:          pagos,
		Estado:         v.Estado,
		ConflictoStock: v.ConflictoStock,
		CreatedAt:      v.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
