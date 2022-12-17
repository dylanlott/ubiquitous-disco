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

## api

the monitor has two main resources so far: monitors and customers.

### monitors api 

the monitors api powers the monitoring and alerting systems.

#### POST /monitors

#### GET /monitors

#### PUT /monitors/:id

#### DELETE /monitors/:id

### customers api

the customers api powers the customer interactions such as subscriptions, purchases, and pricing information.

#### GET /config

Returns the publishable key and the list of prices for the products.

```json
{
    "publishableKey": "",
    "prices": []
}
```

#### POST /create-customer

handleCreateCustomer makes a Stripe customer with the provided email. 
this is the necessary first-step to interacting with our customers API and creates a `customer` cookie with the customer's ID as the value.

Request
```json
{
  "email": ""
}
```

Response
```json
{
    "customer": {
        "address": {
            "city": "",
            "country": "",
            "line1": "",
            "line2": "",
            "postal_code": "",
            "state": ""
        },
        "balance": 0,
        "cash_balance": null,
        "created": 0,
        "currency": "",
        "default_currency": "",
        "default_source": null,
        "deleted": false,
        "delinquent": false,
        "description": "",
        "discount": null,
        "email": "",
        "id": "",
        "invoice_credit_balance": null,
        "invoice_prefix": "",
        "invoice_settings": {
            "custom_fields": null,
            "default_payment_method": null,
            "footer": "",
            "rendering_options": null
        },
        "livemode": false,
        "metadata": {},
        "name": "",
        "next_invoice_sequence": 1,
        "object": "customer",
        "phone": "",
        "preferred_locales": [],
        "shipping": null,
        "sources": null,
        "subscriptions": null,
        "tax": null,
        "tax_exempt": "none",
        "tax_ids": null,
        "test_clock": null
    }
}
```

#### POST /create-subscription

Request
```json
{
  "priceId": ""
}
```

Response 
```json
{
    "subscriptionId": "sub_1MG6e3GxdKKUSt0m9X5e2KGG",
    "clientSecret": "pi_3MG6e3GxdKKUSt0m0DZaSVEq_secret_VVLHq9cPGPOIK8Vg7LW35w6V9"
}
```


## structure and components
`pkg/alerts` - the main alerting library. 
`pkg/db` - contains the PostgreSQL and GORM driver connection and the app's models.
`pkg/server` - contains the server abstraction and handlers that manage the REST API abstraction.

## development
`go run app.go` will run the server locally.

## testing 
`go test -race -v ./...`
