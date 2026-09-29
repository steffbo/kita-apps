package auth

import (
	"context"

	"github.com/google/uuid"
)

type impersonatorKey struct{}

// WithImpersonator marks ctx as a request an admin makes while acting as another user.
func WithImpersonator(ctx context.Context, adminID uuid.UUID) context.Context {
	return context.WithValue(ctx, impersonatorKey{}, adminID)
}

// ImpersonatorFrom returns the admin behind an impersonated request.
func ImpersonatorFrom(ctx context.Context) *uuid.UUID {
	id, ok := ctx.Value(impersonatorKey{}).(uuid.UUID)
	if !ok {
		return nil
	}
	return &id
}
