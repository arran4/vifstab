package rules

import "context"

type targetFstabKey struct{}

func WithTargetFstab(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, targetFstabKey{}, path)
}

func TargetFstabFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if p, ok := ctx.Value(targetFstabKey{}).(string); ok {
		return p
	}
	return ""
}
