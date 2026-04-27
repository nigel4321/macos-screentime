package auth

import "context"

// ctxKey is a private type to avoid collisions with context values
// stored by other packages.
type ctxKey int

const (
	keyAccountID ctxKey = iota + 1
	keyDeviceID
)

// WithAccountID stores the authenticated account id on ctx.
func WithAccountID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyAccountID, id)
}

// AccountIDFromContext returns the authenticated account id, or "" if
// the request did not pass through Authenticator.
func AccountIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keyAccountID).(string)
	return v
}

// WithDeviceID stores the resolved device id on ctx.
func WithDeviceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyDeviceID, id)
}

// DeviceIDFromContext returns the resolved device id, or "" if the
// request did not pass through DeviceContext.
func DeviceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keyDeviceID).(string)
	return v
}
