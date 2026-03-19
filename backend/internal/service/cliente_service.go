package service

import (
	"context"
	"errors"
	"fmt"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ClienteService interface {
	Crear(ctx context.Context, req dto.CrearClienteRequest) (*dto.ClienteResponse, error)
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*dto.ClienteResponse, error)
	Actualizar(ctx context.Context, id uuid.UUID, req dto.UpdateClienteRequest) (*dto.ClienteResponse, error)
	Listar(ctx context.Context, search string, page, limit int) (*dto.ClienteListResponse, error)
	GetDeudores(ctx context.Context) (*dto.ListDeudoresResponse, error)
	GetMovimientos(ctx context.Context, clienteID uuid.UUID, page, limit int) (*dto.MovimientosListResponse, error)
	CargarFiado(ctx context.Context, clienteID uuid.UUID, ventaID uuid.UUID, monto decimal.Decimal) error
	RegistrarPago(ctx context.Context, clienteID uuid.UUID, req dto.RegistrarPagoClienteRequest) (*dto.MovimientoCuentaResponse, error)
}

type clienteService struct {
	repo repository.ClienteRepository
}

func NewClienteService(repo repository.ClienteRepository) ClienteService {
	return &clienteService{repo: repo}
}

func (s *clienteService) Crear(ctx context.Context, req dto.CrearClienteRequest) (*dto.ClienteResponse, error) {
	c := &model.Cliente{
		Nombre:        req.Nombre,
		Telefono:      req.Telefono,
		Email:         req.Email,
		DNI:           req.DNI,
		LimiteCredito: req.LimiteCredito,
		SaldoDeudor:   decimal.Zero,
		Activo:        true,
		Notas:         req.Notas,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return clienteToResponse(c), nil
}

func (s *clienteService) ObtenerPorID(ctx context.Context, id uuid.UUID) (*dto.ClienteResponse, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("cliente no encontrado")
	}
	return clienteToResponse(c), nil
}

func (s *clienteService) Actualizar(ctx context.Context, id uuid.UUID, req dto.UpdateClienteRequest) (*dto.ClienteResponse, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("cliente no encontrado")
	}
	if req.Nombre != nil {
		c.Nombre = *req.Nombre
	}
	if req.Telefono != nil {
		c.Telefono = req.Telefono
	}
	if req.Email != nil {
		c.Email = req.Email
	}
	if req.DNI != nil {
		c.DNI = req.DNI
	}
	if req.LimiteCredito != nil {
		c.LimiteCredito = *req.LimiteCredito
	}
	if req.Activo != nil {
		c.Activo = *req.Activo
	}
	if req.Notas != nil {
		c.Notas = req.Notas
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return clienteToResponse(c), nil
}

func (s *clienteService) Listar(ctx context.Context, search string, page, limit int) (*dto.ClienteListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	clientes, total, err := s.repo.List(ctx, search, page, limit)
	if err != nil {
		return nil, err
	}
	data := make([]dto.ClienteResponse, 0, len(clientes))
	for _, c := range clientes {
		data = append(data, *clienteToResponse(&c))
	}
	return &dto.ClienteListResponse{Data: data, Total: total}, nil
}

func (s *clienteService) GetDeudores(ctx context.Context) (*dto.ListDeudoresResponse, error) {
	clientes, total, err := s.repo.GetDeudores(ctx)
	if err != nil {
		return nil, err
	}
	data := make([]dto.DeudorResponse, 0, len(clientes))
	for _, c := range clientes {
		data = append(data, dto.DeudorResponse{
			ID:            c.ID.String(),
			Nombre:        c.Nombre,
			Telefono:      c.Telefono,
			SaldoDeudor:   c.SaldoDeudor,
			LimiteCredito: c.LimiteCredito,
		})
	}
	return &dto.ListDeudoresResponse{Data: data, Total: total}, nil
}

func (s *clienteService) GetMovimientos(ctx context.Context, clienteID uuid.UUID, page, limit int) (*dto.MovimientosListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	movs, total, err := s.repo.GetMovimientos(ctx, clienteID, page, limit)
	if err != nil {
		return nil, err
	}
	data := make([]dto.MovimientoCuentaResponse, 0, len(movs))
	for _, m := range movs {
		data = append(data, *movimientoToResponse(&m))
	}
	return &dto.MovimientosListResponse{Data: data, Total: total, Page: page, Limit: limit}, nil
}

// CargarFiado creates a "cargo" movement on the customer's credit account.
// Validates that the new saldo does not exceed limite_credito.
func (s *clienteService) CargarFiado(ctx context.Context, clienteID uuid.UUID, ventaID uuid.UUID, monto decimal.Decimal) error {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}

	c, err := s.repo.FindByID(ctx, clienteID)
	if err != nil {
		return errors.New("cliente no encontrado")
	}

	nuevoSaldo := c.SaldoDeudor.Add(monto)
	if c.LimiteCredito.IsPositive() && nuevoSaldo.GreaterThan(c.LimiteCredito) {
		return fmt.Errorf(
			"el monto excede el límite de crédito del cliente (disponible: %s, solicitado: %s)",
			c.LimiteCredito.Sub(c.SaldoDeudor).StringFixed(2),
			monto.StringFixed(2),
		)
	}

	c.SaldoDeudor = nuevoSaldo
	desc := fmt.Sprintf("Cargo por venta fiado")
	refTipo := "venta"
	mov := &model.MovimientoCuenta{
		TenantID:       tid,
		ClienteID:      clienteID,
		Tipo:           "cargo",
		Monto:          monto,
		SaldoPosterior: nuevoSaldo,
		ReferenciaID:   &ventaID,
		ReferenciaTipo: &refTipo,
		Descripcion:    &desc,
	}

	return s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateSaldoTx(tx, c, mov)
	})
}

// RegistrarPago creates a "pago" movement that reduces the customer's debt.
// Validates that pago amount does not exceed current saldo_deudor.
func (s *clienteService) RegistrarPago(ctx context.Context, clienteID uuid.UUID, req dto.RegistrarPagoClienteRequest) (*dto.MovimientoCuentaResponse, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	c, err := s.repo.FindByID(ctx, clienteID)
	if err != nil {
		return nil, errors.New("cliente no encontrado")
	}

	if req.Monto.GreaterThan(c.SaldoDeudor) {
		return nil, fmt.Errorf(
			"el pago (%s) excede el saldo deudor actual (%s)",
			req.Monto.StringFixed(2),
			c.SaldoDeudor.StringFixed(2),
		)
	}

	nuevoSaldo := c.SaldoDeudor.Sub(req.Monto)
	c.SaldoDeudor = nuevoSaldo

	desc := "Pago de cuenta corriente"
	if req.Descripcion != nil && *req.Descripcion != "" {
		desc = *req.Descripcion
	}

	mov := &model.MovimientoCuenta{
		TenantID:       tid,
		ClienteID:      clienteID,
		Tipo:           "pago",
		Monto:          req.Monto,
		SaldoPosterior: nuevoSaldo,
		Descripcion:    &desc,
	}

	txErr := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateSaldoTx(tx, c, mov)
	})
	if txErr != nil {
		return nil, txErr
	}

	return movimientoToResponse(mov), nil
}

// ── Mappers ──────────────────────────────────────────────────────────────────

func clienteToResponse(c *model.Cliente) *dto.ClienteResponse {
	disponible := decimal.Zero
	if c.LimiteCredito.IsPositive() {
		disponible = c.LimiteCredito.Sub(c.SaldoDeudor)
		if disponible.IsNegative() {
			disponible = decimal.Zero
		}
	}
	return &dto.ClienteResponse{
		ID:                c.ID.String(),
		Nombre:            c.Nombre,
		Telefono:          c.Telefono,
		Email:             c.Email,
		DNI:               c.DNI,
		LimiteCredito:     c.LimiteCredito,
		SaldoDeudor:       c.SaldoDeudor,
		CreditoDisponible: disponible,
		Activo:            c.Activo,
		Notas:             c.Notas,
		CreatedAt:         c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         c.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func movimientoToResponse(m *model.MovimientoCuenta) *dto.MovimientoCuentaResponse {
	var refID *string
	if m.ReferenciaID != nil {
		s := m.ReferenciaID.String()
		refID = &s
	}
	return &dto.MovimientoCuentaResponse{
		ID:             m.ID.String(),
		ClienteID:      m.ClienteID.String(),
		Tipo:           m.Tipo,
		Monto:          m.Monto,
		SaldoPosterior: m.SaldoPosterior,
		ReferenciaID:   refID,
		ReferenciaTipo: m.ReferenciaTipo,
		Descripcion:    m.Descripcion,
		CreatedAt:      m.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
