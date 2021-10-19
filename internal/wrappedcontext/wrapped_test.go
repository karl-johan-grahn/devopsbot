package wrappedcontext

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapContextValues(t *testing.T) {
	key := struct{}{}
	cctx, cancel := context.WithCancel(context.Background())
	origCtx := context.WithValue(cctx, key, "test_key")

	ctx := WrapContextValues(context.Background(), origCtx)
	cancel()

	assert.Error(t, origCtx.Err())
	assert.NoError(t, ctx.Err())
	assert.Equal(t, "test_key", ctx.Value(key))
}
