package handler

import (
	"net/http"

	"blendpos/internal/apierror"
	"blendpos/internal/dto"
	"blendpos/internal/middleware"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductosHandler struct{ svc service.ProductoService }

func NewProductosHandler(svc service.ProductoService) *ProductosHandler {
	return &ProductosHandler{svc: svc}
}

func (h *ProductosHandler) Crear(c *gin.Context) {
	var req dto.CrearProductoRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.Crear(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	id, _ := uuid.Parse(resp.ID)
	middleware.AuditLog(c, "create", "producto", &id, map[string]interface{}{"nombre": req.Nombre})
	c.JSON(http.StatusCreated, resp)
}

func (h *ProductosHandler) Listar(c *gin.Context) {
	var filter dto.ProductoFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	resp, err := h.svc.Listar(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar productos"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProductosHandler) ObtenerPorID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	resp, err := h.svc.ObtenerPorID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, apierror.New("Producto no encontrado"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProductosHandler) Actualizar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	var req dto.ActualizarProductoRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.Actualizar(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "update", "producto", &id, req)
	c.JSON(http.StatusOK, resp)
}

func (h *ProductosHandler) Desactivar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	if err := h.svc.Desactivar(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "delete", "producto", &id, nil)
	c.Status(http.StatusNoContent)
}

func (h *ProductosHandler) Reactivar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	if err := h.svc.Reactivar(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ProductosHandler) CrearBulk(c *gin.Context) {
	var req dto.BulkCrearProductosRequest
	if !bindAndValidate(c, &req) {
		return
	}

	results := make([]dto.BulkImportResult, 0, len(req.Productos))
	created := 0
	failed := 0

	for i, p := range req.Productos {
		resp, err := h.svc.Crear(c.Request.Context(), p)
		if err != nil {
			results = append(results, dto.BulkImportResult{
				Index:   i,
				Success: false,
				Error:   err.Error(),
			})
			failed++
		} else {
			results = append(results, dto.BulkImportResult{
				Index:   i,
				Success: true,
				ID:      resp.ID,
			})
			created++
		}
	}

	middleware.AuditLog(c, "bulk_create", "producto", nil, map[string]interface{}{
		"total":   len(req.Productos),
		"created": created,
		"failed":  failed,
	})

	c.JSON(http.StatusOK, dto.BulkImportResponse{
		Total:   len(req.Productos),
		Created: created,
		Failed:  failed,
		Results: results,
	})
}

func (h *ProductosHandler) AjustarStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	var req dto.AjustarStockRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.AjustarStock(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// CrearVariante creates a variant (child) product from a parent product.
// POST /v1/productos/:id/variantes
func (h *ProductosHandler) CrearVariante(c *gin.Context) {
	padreID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	var req dto.CrearVarianteRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.CrearVariante(c.Request.Context(), padreID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	id, _ := uuid.Parse(resp.ID)
	middleware.AuditLog(c, "create_variant", "producto", &id, map[string]interface{}{
		"padre_id":  padreID.String(),
		"atributos": req.Atributos,
	})
	c.JSON(http.StatusCreated, resp)
}

// ListarVariantes returns all variants of a parent product.
// GET /v1/productos/:id/variantes
func (h *ProductosHandler) ListarVariantes(c *gin.Context) {
	padreID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	variantes, err := h.svc.ListarVariantes(c.Request.Context(), padreID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, variantes)
}
