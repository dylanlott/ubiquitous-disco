package alerts

import (
	"context"
	"fmt"
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
// * Alerts can be called multiple times.
// * Alerts can't fail, by design, but they could log if needed.
type Alert func(ctx context.Context)

// Check runs and returns a bool and an error.
// * A check can pass and still return an error, e.g. degraded service
type Check func(ctx context.Context) (bool, error)

// Siren is responsible for starting, restarting, and stopping
// a set of Monitors. It interacts with the HTTP wrapper to become the
// Monitors resource.
type Siren struct {
	sync.Mutex
	Monitors []*Monitor
}

// Add adds a Monitor to the Siren and starts the Monitor.
func (s *Siren) Add(ctx context.Context, mon *Monitor) error {
	s.Lock()
	s.Monitors = append(s.Monitors, mon)
	s.Unlock()

	go mon.Run(ctx)

	return nil
}

// Run starts a monitor and listens for context cancellations
func (m *Monitor) Run(ctx context.Context) error {
	// initialize the monitor
	if err := m.Init(); err != nil {
		return fmt.Errorf("failed to initialize: %+v", err)
	}

	// start running the monitor
	for {
		// run check once at the beginning and then every mon.Interval
		ok, err := m.Check(ctx)
		if !ok {
			// alert when check fails
			m.Alert(ctx)
			// break out of the loop and return our reason for failure
			return err
		}
		time.Sleep(m.Interval)
	}
}

// Init validates the monitor's configuration and prepares it for execution.
func (m *Monitor) Init() error {
	// for right now, this just returns nil
	return nil
}