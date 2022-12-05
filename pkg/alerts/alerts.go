package alerts

import (
	"context"
	"sync"
	"time"
)

// Alerts is a package for creating a monitoring a set of Monitors.
// A Monitor has an Alert and a Check on it.
// A Check is run at an interval decided by the Monitor.
// A Check func runs and returns a bool and an error.
// A fail on the bool will trigger a call of the Alert function.

// Influx DB documentation is at
// * https://docs.influxdata.com/influxdb/cloud/api-guide/client-libraries/go/

// Monitor combines an Alert, a Check, and an Interval.
// It has a single method Run that calls Check at every Interval.
type Monitor struct {
	Alert    Alert
	Check    Check
	Interval time.Duration
}

// Alert is called when a Check returns false.
// * Alerts take a context and an error which gives access to requestIDs
// and tracing functions as well as the cause of the failed check.
// * Alerts may be called multiple times, but are usually called once.
// * Alerts can't fail, by design. If they're unsuccessful in creating
// their notification, that must be determined by logs.
type Alert func(ctx context.Context, err error)

// Check runs and returns a bool and an error.
// * A check can pass and still return an error, e.g. degraded service.
type Check func(ctx context.Context) (bool, error)

// Siren is responsible for starting, restarting, and stopping
// a set of Monitors. It interacts with the HTTP wrapper to become the
// Monitors resource.
type Siren struct {
	sync.Mutex
	monitors []*Monitor
}

// Add adds a Monitor to the Siren and starts the Monitor.
func (s *Siren) Add(ctx context.Context, mon *Monitor) error {
	s.Lock()
	s.monitors = append(s.monitors, mon)
	s.Unlock()

	go mon.Run(ctx)

	return nil
}

// Run starts a monitor and listens for context cancellations
// TODO: consider exponential backoff upon failed checks
func (m *Monitor) Run(ctx context.Context) error {
	// start running the monitor
	for {
		// run check once at the beginning and then every mon.Interval
		ok, err := m.Check(ctx)
		if !ok {
			// alert when check fails
			go m.Alert(ctx, err)
		}
		time.Sleep(m.Interval)
	}
}
