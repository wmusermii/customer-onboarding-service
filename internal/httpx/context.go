package httpx

import "context"

type ctxKey int

const (
	keyRequestID ctxKey = iota
	keySubject
	keyChannel
)

func WithRequestID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, keyRequestID, v)
}
func RequestID(ctx context.Context) string { v, _ := ctx.Value(keyRequestID).(string); return v }

func WithSubject(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, keySubject, v)
}
func Subject(ctx context.Context) string { v, _ := ctx.Value(keySubject).(string); return v }

func WithChannel(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, keyChannel, v)
}
func Channel(ctx context.Context) string { v, _ := ctx.Value(keyChannel).(string); return v }
