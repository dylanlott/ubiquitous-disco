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

## customers api

the customers api powers the customer interactions such as subscriptions, purchases, and pricing information.

### subscribing a customer

customers are subscribed to the system with combination of calls to Stripe and our own server. 
we first must create a customer by POSTing to `/create-customer` and then we use the customer ID returned from that call to POST to `/create-subscription` which creates a Subscription object for the customer that is tied to their customer ID. 

by this point, we are setup to bill them on a monthly basis but the custoemr has not yet been charged. 
to charge them, the client confirms the payment with the secret and subscription ID returned by the `/create-subscription` request which will charge the card info provided for the subscription. 

if they have ordered monitor devices, that is to say if the quantity is greater than 0, they will then be charged for the appropriate number of devices.

> NB: The device price ID and the subscription ID are currently hard-coded.

- step 1. create a customer - POST /create-customer
  - this creates a cookie with the customer ID
- step 2. create a subscription - POST /create-subscription
  - this returns a clientSecret and a subscription ID which is necessary when confirming the payment in step 3.
- step 3. confirm payment for subscription - `handleConfirmPayment` in `register.html`
  - this charges the card info provided for the subscription.
- step 4. confirm payment for devices - `/charge` request in `register.html`
  - this charges the card info for the quantity of devices ordered.
  - if 0 are ordered, it does not get called.

**For testing Stripe payments**:
- Try the successful test card: `4242424242424242`.
- Try the test card that requires SCA: `4000002500003155`.
- Use any _future_ expiry date, CVC, and 5 digit postal code.

### GET /config

Returns the publishable key and the list of prices for the products.

Response
```json
{
    "publishableKey": "",
    "prices": []
}
```

### POST /create-customer

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

### POST /create-subscription

Request
```json
{
  "priceId": ""
}
```

Response 
```json
{
    "subscriptionId": "sub_xxx",
    "clientSecret": "pi_xxx"
}
```

### POST /charge

Request 
```json
{
  "token": "",
  "quantity": 1
}
```

Response

Returns a Stripe charge object.

```json
{
  "id": "ch_XXX",
  "object": "charge",
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
