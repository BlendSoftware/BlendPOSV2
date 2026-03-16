// Package tenantctx provides a zero-dependency context key and helper for
// propagating the tenant UUID through the request context.
//
// It is intentionally a leaf package (imports only stdlib + uuid) so that
// both middleware and repository can import it without creating an import cycle.
package tenantctx

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type contextKey string

// Key is the context key under which the tenant UUID is stored.
// Using an exported constant lets middleware set it and repository read it
// without either package importing the other.
const Key contextKey = "tenant_id"

// FromContext retrieves the tenant UUID injected by TenantMiddleware.
// Returns an error if the context was not enriched (e.g. unauthenticated path).
func FromContext(ctx context.Context) (uuid.UUID, error) {
	tid, ok := ctx.Value(Key).(uuid.UUID)
	if !ok || tid == uuid.Nil {
		return uuid.Nil, errors.New("tenant_id not found in context — is TenantMiddleware active?")
	}
	return tid, nil
}
