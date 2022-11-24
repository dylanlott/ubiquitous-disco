package alerts

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
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

// Alerts are called when a Check returns false.
// Alerts can be called multiple times.
// Alerts can't fail, by design, but they could log if needed.
type Alert func(ctx context.Context)

// Checks run and return a bool and an error.
// A check can pass and still return an error.
type Check func(ctx context.Context) (bool, error)

// Siren is responsible for starting, restarting, and stopping
// a set of Monitors.
type Siren struct {
	sync.Mutex
	Monitors []*Monitor
}

// Adds a Monitor to the Siren.
func (s *Siren) Add(ctx context.Context, mon *Monitor) error {
	s.Lock()
	s.Monitors = append(s.Monitors, mon)
	s.Unlock()

	go mon.Run(ctx)

	return nil
}

// Run starts a monitor and listens for context cancellations
func (m *Monitor) Run(ctx context.Context) error {
	for {
		// run check every mon.Interval
		time.Sleep(m.Interval)
		ok, err := m.Check(ctx)
		if !ok {
			// alert when check fails
			m.Alert(ctx)
			// break out of the loop and return our reason for failure
			return err
		}
	}
}

///////////////////////////
// INFLUXDB IMPLEMENTATION
///////////////////////////

// InfluxClient holds methods for querying it and creating Alerts.
type InfluxClient struct {
	client influxdb2.Client
}

// Creates a new InfluxDB client or returns an error.
func NewInfluxClient(ctx context.Context) (*InfluxClient, error) {
	influxURL := os.Getenv("INFLUX_URL")
	influxToken := os.Getenv("INFLUX_TOKEN")
	client := influxdb2.NewClient(influxURL, influxToken)

	go func(c context.Context) {
		<-c.Done()
		log.Printf("context cancellation detected")
		// TODO: Handle client closure correctly
		// defer client.Close()
	}(ctx)

	return &InfluxClient{
		client: client,
	}, nil
}

// Create makes a new Monitor on the given DataSource.
func (i *InfluxClient) Create(ctx context.Context, query string) (*Monitor, error) {
	org := ""
	api := i.client.QueryAPI(org)
	api.Query(ctx, query)
	m := &Monitor{
		Alert: func(ctx context.Context) {
			log.Printf("alerted")
		},
		Check: func(ctx context.Context) (bool, error) {
			return false, fmt.Errorf("not impl")
		},
	}
	return m, nil
}
