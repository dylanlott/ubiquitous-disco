package alerts

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

///////////////////////////
// INFLUXDB IMPLEMENTATION
///////////////////////////

// InfluxClient holds methods for querying it and creating Alerts.
type InfluxClient struct {
	client influxdb2.Client
}

// NewInfluxClient creates a new InfluxDB client or returns an error.
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

// create makes a new Monitor on the given DataSource.
func (i *InfluxClient) create(ctx context.Context, query string) (*Monitor, error) {
	// pass the client the oragnizationID must be
	orgID := os.Getenv("INFLUX_ORGID")
	if orgID == "" {
		return nil, fmt.Errorf("ErrInvalidOrgID")
	}
	api := i.client.QueryAPI(orgID) // TODO: get from env?
	m := &Monitor{
		Alert: func(ctx context.Context, err error) {
			log.Printf("ERROR: monitor alerted: %+v", err)
		},
		Interval: time.Minute * 15,
		Check: func(ctx context.Context) (bool, error) {
			result, err := api.Query(ctx, query)
			if err != nil {
				log.Printf("ERROR QUERYING INFLUXDB %+v", err)
			}
			defer result.Close()

			// loop over until we prove our monitor correct.
			// TODO: check some configurable upper and lower bounds.
			ok := false
			for result.Next() {
				r := result.Record()
				// fmt.Printf("r.Values(): %v\n", r.Values())
				// fmt.Printf("r.Result(): %v\n", r.Result())
				// fmt.Printf("r.Table(): %v\n", r.Table())
				// fmt.Printf("r.Value(): %v\n", r.Value())
				// fmt.Printf("r.Time(): %v\n", r.Time())
				fmt.Printf("r.Values(): %v\n", r.Values())
				then := time.Now().Add(-time.Minute * 15)
				if r.Start().After(then) {
					ok = true
				}
			}
			return ok, err
		},
	}
	return m, nil
}
