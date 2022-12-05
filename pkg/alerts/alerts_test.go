package alerts

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/matryer/is"
)

func TestAlerts(t *testing.T) {
	t.Run("should add and start an influx monitor", func(t *testing.T) {
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

		s := &Siren{
			monitors: []*Monitor{},
		}

		err = s.Add(ctx, mon)
		is.NoErr(err)
		is.Equal(len(s.monitors), 1)
	})

	t.Run("should call alert on fail", func(t *testing.T) {
		is := is.New(t)
		ctx := context.Background()

		wg := sync.WaitGroup{}
		called := 0
		mon := &Monitor{
			Alert: func(ctx context.Context, err error) {
				called++
				is.True(err.Error() == "ErrMock")
				wg.Done()
			},
			Check: func(ctx context.Context) (bool, error) {
				wg.Add(1)
				return false, fmt.Errorf("ErrMock")
			},
		}

		s := &Siren{
			monitors: []*Monitor{},
		}

		err := s.Add(ctx, mon)
		is.NoErr(err)
		wg.Wait()
	})
}
