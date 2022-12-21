package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/charge"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/invoice"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/price"
	"github.com/stripe/stripe-go/v72/sub"
	"github.com/stripe/stripe-go/v72/webhook"
)

// init loads the stripe key for this file at server start.
func init() {
	// NB: Do not attach to S so that we never risk exposing keys
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}

// UNIT_PRICE should eventually be fetched from Stripe but holds the unit price of
// GRO-01 Monitors
const UNIT_PRICE = 15000 // $150 per piece USD

// NB: subscriptions uses cookies to track users and will need to be wired into
// whatever we design for authentication

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	params := &stripe.PriceListParams{} // empty params means fetch all
	prices := make([]*stripe.Price, 0)  // the Price is all of the info for the product
	i := price.List(params)
	for i.Next() {
		prices = append(prices, i.Price())
	}

	writeJSON(w, struct {
		PublishableKey string          `json:"publishableKey"`
		Prices         []*stripe.Price `json:"prices"`
	}{
		PublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		Prices:         prices,
	}, nil)
}

// handleCreateCustomer makes a Stripe customer with the provided email.
// This is the necessary first-step to interacting with our customers API
// and creates a cookie
func handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		// Name     string `json:"name"`
		Email string `json:"email"`
		// Phone    string `json:"phone"`
		Password string `json:"password"` // TODO: add password handling here
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, nil, err)
		return
	}

	// TODO: Check that they're not already registered in our system with email
	// TODO: add password handling and user generation here

	// make a basic 3 field customer in stripe
	params := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
		// Name:  stripe.String(req.Name),
		// Phone: stripe.String(req.Phone),
	}

	c, err := customer.New(params)
	if err != nil {
		writeJSON(w, nil, err)
		return
	}

	// You should store the ID of the customer in your database alongside your
	// users. This sample uses cookies to simulate auth.
	http.SetCookie(w, &http.Cookie{
		Name:  "customer",
		Value: c.ID,
	})

	writeJSON(w, struct {
		Customer *stripe.Customer `json:"customer"`
	}{
		Customer: c,
	}, nil)
}

// handleCreateSubscription creates a subscription for the customer that the `customer` cookie specifies.
func handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PriceID string `json:"priceId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, nil, err)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	// Read customer from cookie to simulate auth
	cookie, _ := r.Cookie("customer")
	customerID := cookie.Value

	// Create subscription
	subscriptionParams := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(req.PriceID),
			},
		},
		PaymentBehavior: stripe.String("default_incomplete"),
	}
	subscriptionParams.AddExpand("latest_invoice.payment_intent")

	s, err := sub.New(subscriptionParams)
	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("sub.New: %v", err)
		return
	}

	writeJSON(w, struct {
		SubscriptionID string `json:"subscriptionId"`
		ClientSecret   string `json:"clientSecret"`
	}{
		SubscriptionID: s.ID,
		ClientSecret:   s.LatestInvoice.PaymentIntent.ClientSecret,
	}, nil)
}

func handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SubscriptionID string `json:"subscriptionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, nil, err)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	s, err := sub.Cancel(req.SubscriptionID, nil)

	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("sub.Cancel: %v", err)
		return
	}

	writeJSON(w, struct {
		Subscription *stripe.Subscription `json:"subscription"`
	}{
		Subscription: s,
	}, nil)
}

func handleInvoicePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	// Read customer from cookie to simulate auth
	cookie, err := r.Cookie("customer")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	customerID := cookie.Value

	query := r.URL.Query()
	subscriptionID := query.Get("subscriptionId")
	newPriceLookupKey := strings.ToUpper(query.Get("newPriceLookupKey"))

	s, err := sub.Get(subscriptionID, nil)
	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("sub.Get: %v", err)
		return
	}
	params := &stripe.InvoiceParams{
		Customer:     stripe.String(customerID),
		Subscription: stripe.String(subscriptionID),
		SubscriptionItems: []*stripe.SubscriptionItemsParams{{
			ID:    stripe.String(s.Items.Data[0].ID),
			Price: stripe.String(os.Getenv(newPriceLookupKey)),
		}},
	}
	in, err := invoice.GetNext(params)

	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("invoice.GetNext: %v", err)
		return
	}

	writeJSON(w, struct {
		Invoice *stripe.Invoice `json:"invoice"`
	}{
		Invoice: in,
	}, nil)
}

func handleUpdateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SubscriptionID    string `json:"subscriptionId"`
		NewPriceLookupKey string `json:"newPriceLookupKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, nil, err)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	// This is the ID of the Stripe Price object to which the subscription
	// will be upgraded or downgraded.
	newPriceID := os.Getenv(strings.ToUpper(req.NewPriceLookupKey))

	// Fetch the subscription to access the related subscription item's ID
	// that will be updated. In practice, you might want to store the
	// Subscription Item ID in your database to avoid this API call.
	s, err := sub.Get(req.SubscriptionID, nil)
	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("sub.Get: %v", err)
		return
	}

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{{
			ID:    stripe.String(s.Items.Data[0].ID),
			Price: stripe.String(newPriceID),
		}},
	}

	updatedSubscription, err := sub.Update(req.SubscriptionID, params)

	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("sub.Update: %v", err)
		return
	}

	writeJSON(w, struct {
		Subscription *stripe.Subscription `json:"subscription"`
	}{
		Subscription: updatedSubscription,
	}, nil)

}

func handleListSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	// Read customer from cookie to simulate auth
	cookie, err := r.Cookie("customer")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	customerID := cookie.Value

	params := &stripe.SubscriptionListParams{
		Customer: customerID,
		Status:   "all",
	}
	params.AddExpand("data.default_payment_method")
	i := sub.List(params)

	writeJSON(w, struct {
		Subscriptions *stripe.SubscriptionList `json:"subscriptions"`
	}{
		Subscriptions: i.SubscriptionList(),
	}, nil)
}

// handleCharge is pinged by the Checkout route's credit card form.
// TODO: make handleCharge use request body for handling amounts and quantities
func handleCharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Quantity string `json:"quantity"`
		Token    string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, nil, err)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	log.Printf("charge request: %+v", req)

	qty, err := strconv.Atoi(req.Quantity)
	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("failed to parse quantity: %v", err)
		return
	}

	total := qty * UNIT_PRICE
	params := &stripe.ChargeParams{
		Amount:      stripe.Int64(int64(total)),
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Description: stripe.String("GRO-01 Monitor"),
	}
	params.SetSource(req.Token)

	ch, err := charge.New(params)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("failed to charge: %+v", err)))
		log.Printf("failed to charge card: %+v", err)
		return
	}
	writeJSON(w, ch, nil)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("ioutil.ReadAll: %v", err)
		return
	}

	event, err := webhook.ConstructEvent(b, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		writeJSON(w, nil, err)
		log.Printf("webhook.ConstructEvent: %v", err)
		return
	}

	if event.Type == "invoice.payment_succeeded" {
		var invoice stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		pi, _ := paymentintent.Get(
			invoice.PaymentIntent.ID,
			nil,
		)

		params := &stripe.SubscriptionParams{
			DefaultPaymentMethod: stripe.String(pi.PaymentMethod.ID),
		}
		sub.Update(invoice.Subscription.ID, params)
		fmt.Println("Default payment method set for subscription: ", pi.PaymentMethod)
	}
	fmt.Println("Payment succeeded for invoice: ", event.ID)
}

type errResp struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, v interface{}, err error) {
	var respVal interface{}
	if err != nil {
		msg := err.Error()
		var serr *stripe.Error
		if errors.As(err, &serr) {
			msg = serr.Msg
		}
		w.WriteHeader(http.StatusBadRequest)
		var e errResp
		e.Error.Message = msg
		respVal = e
	} else {
		respVal = v
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(respVal); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewEncoder.Encode: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, &buf); err != nil {
		log.Printf("io.Copy: %v", err)
		return
	}
}
