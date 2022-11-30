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

	query := `from(bucket: "growmon")
	|> range(start: -10000, stop: now())
	|> filter(fn: (r) => r["_measurement"] == "STBProto")
	|> filter(fn: (r) => r["_field"] == "heat_index" or r["_field"] == "humidity" or r["_field"] == "temperature" or r["_field"] == "uuid")
	|> filter(fn: (r) => r["UUID"] == "UUID: 00-00-01")
	|> aggregateWindow(every: 30m, fn: mean, createEmpty: false)
	|> yield(name: "mean")`

	mon, err := ic.create(ctx, query)
	is.NoErr(err)

	is.True(mon != nil)
	err = mon.Run(ctx)
	is.NoErr(err)
}
