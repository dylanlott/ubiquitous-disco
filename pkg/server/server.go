package server

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"

	"github.com/dylanlott/ubiquitous-disco/pkg/alerts"
	"github.com/dylanlott/ubiquitous-disco/pkg/db"
)

// key is used tracking context keys
type key int

// requestIDFunc should return a unique ID to key requests for tracing
type requestIDFunc func() string

const (
	// requestIDKey is used for tracing
	requestIDKey key = 0
)

// nextRequestID is an anonymous function that returns a unique string ID for requests
var nextRequestID = func() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// S holds all of the relevant pieces together
// for our monitoring service.
type S struct {
	db     *gorm.DB
	influx influxdb2.Client
	siren  *alerts.Siren
	srv    *http.Server
}

// New creates a new server with a given set of templates and static
// embeds and returns it
func New(t *template.Template, static embed.FS, addr string) (*S, error) {
	s := &S{
		db: db.New(),
		srv: &http.Server{
			Addr: addr,
		},
	}

	// connect to influx
	influxURL := os.Getenv("INFLUX_URL")
	influxToken := os.Getenv("INFLUX_TOKEN")
	client := influxdb2.NewClient(influxURL, influxToken)
	defer client.Close()
	s.influx = client

	// make a new logger
	logger := log.New(os.Stdout, "api: ", log.LstdFlags)

	// start a new http server with logging and tracing
	s.srv.Handler = tracing(nextRequestID)(logging(logger)(s.routes(t, static)))

	// connect to postgres through gorm
	return s, nil
}

// Serve listens at the configured address
// TODO: Handle context cancellation and graceful shutdown
func (s *S) Serve() error {
	log.Printf("listening at %s", s.srv.Addr)
	return s.srv.ListenAndServe()
}

// routes muxes the templates with the handlers and returns the muxer
func (s *S) routes(t *template.Template, static embed.FS) *mux.Router {
	router := mux.NewRouter()

	// serve static files
	var staticFS = http.FS(static)
	router.PathPrefix("/static").Handler(http.FileServer(staticFS))

	// index
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hc, err := s.influx.Health(context.Background())
		if err != nil {
			log.Fatalf("failed influxDB health check: %v", err)
		}

		data := map[string]string{
			"Region": os.Getenv("FLY_REGION"),
			"Status": string(hc.Status),
		}
		t.ExecuteTemplate(w, "index.html.tmpl", data)
	})

	router.Handle("/metrics", promhttp.Handler())

	// product handles simple product pages
	router.HandleFunc("/product", func(w http.ResponseWriter, r *http.Request) {
		t.ExecuteTemplate(w, "product.html.tmpl", nil)
	})

	// checkout serves the credit card form
	router.HandleFunc("/checkout", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{}
		t.ExecuteTemplate(w, "checkout.html.tmpl", data)
	})

	router.HandleFunc("/buckets", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{}
		buckets, err := s.influx.BucketsAPI().GetBuckets(context.Background())
		if err != nil {
			w.Write([]byte(fmt.Sprintf("failed to get buckets: %s", err)))
			return
		}
		for _, b := range *buckets {
			data[b.Name] = b
		}
		t.ExecuteTemplate(w, "buckets.html.tmpl", map[string]interface{}{"Buckets": data})
	})

	// monitors
	router.HandleFunc("/monitors", s.monitorHandler)
	router.HandleFunc("/monitors/{id}", s.monitorHandler)

	// customers
	router.HandleFunc("/config", handleConfig)
	router.HandleFunc("/premium", func(w http.ResponseWriter, r *http.Request) {
		t.ExecuteTemplate(w, "premium.html", nil)
	})
	router.HandleFunc("/create-customer", handleCreateCustomer)
	router.HandleFunc("/create-subscription", handleCreateSubscription)
	router.HandleFunc("/cancel-subscription", handleCancelSubscription)
	router.HandleFunc("/update-subscription", handleUpdateSubscription)
	router.HandleFunc("/invoice-preview", handleInvoicePreview)
	router.HandleFunc("/subscriptions", handleListSubscriptions)
	router.HandleFunc("/webhook", handleWebhook)
	router.HandleFunc("/charge", handleCharge)

	return router
}

// tracing adds tracing to our API by wrapping requests and adding an X-Request-ID header.
func tracing(nextRequestID requestIDFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// logging wraps the request handlers in a logger with the provided requestID
func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}
