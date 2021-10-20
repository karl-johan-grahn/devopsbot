package wrappedcontext

import "context"

// WrapContextValues returns a Context that has the same cancellation and deadline
// behavior as the first argument but has values from the second argument. This
// is useful when you need to create a context with a longer deadline than the
// one you were called with, but still want the values present in the context.
func WrapContextValues(ctx, values context.Context) context.Context {
	return wrappedCtx{Context: ctx, values: values}
}

type wrappedCtx struct {
	context.Context
	values context.Context
}

func (c wrappedCtx) Value(key interface{}) interface{} {
	return c.values.Value(key)
}
