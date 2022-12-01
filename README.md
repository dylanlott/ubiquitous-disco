# growalert

> a monitoring and alerting library that uses InfluxDB and PostgreSQL to monitor timeseries data.

## alerting
`pkg/alerts` holds the alerting library. It manages an internal collection of monitors that run checks on a configurable interval. It should have no external dependencies and should not let the implementation logic of any source change its structure.

It defines two main types - the Alert and the Check function. 

```go
// Alert is called when a Check returns false.
// * Alerts may be called multiple times, but are usually called once.
// * Alerts can't fail, by design, but they could log if needed.
type Alert func(ctx context.Context, err ...error)

  // Check runs and returns a bool and an error.
// * A check can pass and still return an error, e.g. degraded service
type Check func(ctx context.Context) (bool, error)
```

The Monitor struct ties these two together with an Interval function. A Monitor runs its Check function every Interval and calls the Alert whenever it fails.

## structure and components
`pkg/alerts` - the main alerting library. 
`pkg/db` - contains the PostgreSQL and GORM driver connection and the app's models.
`pkg/server` - contains the server abstraction and handlers that manage the REST API abstraction.

## development
`go run app.go` will run the server locally.

## testing 
`go test -race -v ./...`
