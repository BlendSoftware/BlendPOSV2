package middleware

import (
	"context"
	"reflect"
	"sync/atomic"
	"time"

	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// violationCtxKey is the context key for the atomic violation counter.
type violationCtxKeyType string

const violationCtxKey violationCtxKeyType = "tenant_audit_violations"

// ---------------------------------------------------------------------------
// 1. GORM Callback — verifies tenant_id on query results
// ---------------------------------------------------------------------------

// RegisterTenantAuditCallback registers a GORM callback that fires after every
// SELECT. It inspects the query result via reflection: if the destination struct
// has a TenantID field whose value differs from the tenant_id in the request
// context, it logs a CRITICAL alert and increments the violation counter.
//
// Performance: uses reflect only on the outermost type (no deep recursion).
// Overhead is negligible (< 100ns per row check on cache-hot structs).
func RegisterTenantAuditCallback(db *gorm.DB) {
	db.Callback().Query().After("gorm:query").Register("tenant:audit", tenantAuditCallback)
}

func tenantAuditCallback(db *gorm.DB) {
	if db.Statement == nil || db.Statement.Context == nil {
		return
	}

	ctx := db.Statement.Context

	// If there is no tenant in the context this is a public/system query — skip.
	expectedTID, err := tenantctx.FromContext(ctx)
	if err != nil {
		return
	}

	dest := db.Statement.Dest
	if dest == nil {
		return
	}

	// Check every row in the result set.
	violations := checkDestTenantID(dest, expectedTID)
	if violations == 0 {
		return
	}

	// Increment the request-scoped atomic counter.
	if counter, ok := ctx.Value(violationCtxKey).(*atomic.Int64); ok && counter != nil {
		counter.Add(int64(violations))
	}

	log.Error().
		Str("level", "CRITICAL").
		Str("expected_tenant_id", expectedTID.String()).
		Int("violations", violations).
		Str("table", db.Statement.Table).
		Str("sql", db.Statement.SQL.String()).
		Msg("TENANT ISOLATION VIOLATION: query returned rows belonging to a different tenant")
}

// checkDestTenantID inspects the GORM destination value and returns the number
// of rows whose TenantID does not match expectedTID. Supports single structs,
// pointers, and slices of structs.
func checkDestTenantID(dest interface{}, expectedTID uuid.UUID) int {
	v := reflect.ValueOf(dest)
	if !v.IsValid() {
		return 0
	}

	// Dereference pointer(s).
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice:
		violations := 0
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			for elem.Kind() == reflect.Ptr {
				if elem.IsNil() {
					break
				}
				elem = elem.Elem()
			}
			if elem.Kind() == reflect.Struct {
				if mismatch(elem, expectedTID) {
					violations++
				}
			}
		}
		return violations

	case reflect.Struct:
		if mismatch(v, expectedTID) {
			return 1
		}
	}

	return 0
}

// mismatch returns true if the struct has a TenantID field with a UUID value
// that does NOT match expectedTID. Returns false if the field does not exist
// or is zero-valued (e.g. not populated by the query).
func mismatch(v reflect.Value, expectedTID uuid.UUID) bool {
	f := v.FieldByName("TenantID")
	if !f.IsValid() {
		return false
	}

	// Support both uuid.UUID ([16]byte) and string representations.
	switch f.Type() {
	case reflect.TypeOf(uuid.UUID{}):
		rowTID, ok := f.Interface().(uuid.UUID)
		if !ok || rowTID == uuid.Nil {
			return false
		}
		return rowTID != expectedTID

	case reflect.TypeOf(""):
		s := f.String()
		if s == "" {
			return false
		}
		rowTID, err := uuid.Parse(s)
		if err != nil || rowTID == uuid.Nil {
			return false
		}
		return rowTID != expectedTID
	}

	return false
}

// ---------------------------------------------------------------------------
// 2. HTTP Middleware — audit log + violation summary
// ---------------------------------------------------------------------------

// TenantAuditMiddleware logs every authenticated request with tenant context
// and checks whether any tenant isolation violations were detected during the
// request lifecycle (via the GORM callback above).
//
// Place it AFTER TenantMiddleware in the middleware chain.
func TenantAuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Inject an atomic counter into the request context so the GORM
		// callback can increment it without locks.
		var counter atomic.Int64
		ctx := context.WithValue(c.Request.Context(), violationCtxKey, &counter)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		latency := time.Since(start)

		tenantID := ""
		userID := ""
		role := ""
		if v, exists := c.Get(ClaimsKey); exists {
			if claims, ok := v.(*JWTClaims); ok && claims != nil {
				tenantID = claims.TenantID
				userID = claims.UserID
				role = claims.Rol
			}
		}

		violations := counter.Load()

		if violations > 0 {
			log.Error().
				Str("level", "CRITICAL").
				Str("tenant_id", tenantID).
				Str("user_id", userID).
				Str("role", role).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Int("status", c.Writer.Status()).
				Dur("latency", latency).
				Str("request_id", c.GetString(RequestIDKey)).
				Int64("tenant_violations", violations).
				Msg("TENANT AUDIT CRITICAL: isolation violations detected in request")
		} else {
			log.Debug().
				Str("tenant_id", tenantID).
				Str("user_id", userID).
				Str("role", role).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Int("status", c.Writer.Status()).
				Dur("latency", latency).
				Str("request_id", c.GetString(RequestIDKey)).
				Msg("tenant_audit")
		}
	}
}

// ViolationCount returns the number of tenant isolation violations detected
// during the current request. Useful for tests and health checks.
func ViolationCount(ctx context.Context) int64 {
	if counter, ok := ctx.Value(violationCtxKey).(*atomic.Int64); ok && counter != nil {
		return counter.Load()
	}
	return 0
}
