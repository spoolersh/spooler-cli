package main

import (
	"context"
	"log/slog"
)

type contextKey[T any] struct{}

func with[T any](p context.Context, v T) context.Context {
	return context.WithValue(p, contextKey[T]{}, v)
}
func from[T any](p context.Context) T {
	t, _ := p.Value(contextKey[T]{}).(T)
	return t
}

func contextConfig(ctx context.Context) *config {
	return from[*config](ctx)
}
func contextLogger(ctx context.Context) *slog.Logger {
	return from[*slog.Logger](ctx)
}
