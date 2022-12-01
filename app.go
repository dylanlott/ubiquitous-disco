package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/fly-apps/go-example/pkg/db"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/stripe/stripe-go/v73"
	"github.com/stripe/stripe-go/v73/charge"
)

// Config holds config values for the application
type Config struct {
	Port        string
	InfluxURL   string
	InfluxToken string
	TemplateDir string
}

//go:embed templates/*
var resources embed.FS

var t = template.Must(template.ParseFS(resources, "templates/*"))

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	gdb := db.New()
	log.Printf("gdb: %+v", gdb)

	influxURL := os.Getenv("INFLUX_URL")
	influxToken := os.Getenv("INFLUX_TOKEN")
	client := influxdb2.NewClient(influxURL, influxToken)
	defer client.Close()

	hc, err := client.Health(context.Background())
	if err != nil {
		log.Fatalf("failed influxDB health check: %v", err)
	}

	// serves the home page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"Region": os.Getenv("FLY_REGION"),
			"Status": string(hc.Status),
		}
		t.ExecuteTemplate(w, "index.html.tmpl", data)
	})

	// charge is pinged by the Checkout route's credit card form
	http.HandleFunc("/charge", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("charge hit: %+v", r.Form)
		stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

		// Token is created using Stripe Checkout or Elements!
		// Get the payment token ID submitted by the form:
		token := r.FormValue("stripeToken")

		params := &stripe.ChargeParams{
			// TODO: charge for correct amount
			Amount: stripe.Int64(999),
			// TODO: charge for quantity to allow multiple unit orders
			Currency:    stripe.String(string(stripe.CurrencyUSD)),
			Description: stripe.String("Example charge"),
		}
		params.SetSource(token)

		ch, err := charge.New(params)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("failed to charge: %+v", err)))
			return
		}

		log.Printf("successfully charged: %+v", ch)

		// redirect to ch.ReceiptURL
		http.Redirect(w, r, ch.ReceiptURL, http.StatusMovedPermanently)
	})

	// checkout serves the credit card form
	http.HandleFunc("/checkout", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"Region": os.Getenv("FLY_REGION"),
			"Status": string(hc.Status),
		}
		t.ExecuteTemplate(w, "checkout.html.tmpl", data)
	})

	http.HandleFunc("/buckets", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{}
		buckets, err := client.BucketsAPI().GetBuckets(context.Background())
		if err != nil {
			w.Write([]byte(fmt.Sprintf("failed to get buckets: %s", err)))
			return
		}
		for _, b := range *buckets {
			data[b.Name] = b
		}
		t.ExecuteTemplate(w, "buckets.html.tmpl", map[string]interface{}{"Buckets": data})
	})

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
