package alerts

import (
	"context"
	"testing"

	"github.com/matryer/is"
)

func TestAlerts(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	ic, err := NewInfluxClient(ctx)
	is.NoErr(err)
	query := "TODO"
	mon, err := ic.Create(ctx, query)
	is.NoErr(err)
	is.True(mon != nil)
	err = mon.Run(ctx)
	is.NoErr(err)
}
