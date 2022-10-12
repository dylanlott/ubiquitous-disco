package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

//go:embed templates/*
var resources embed.FS

var t = template.Must(template.ParseFS(resources, "templates/*"))

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	influxURL := os.Getenv("INFLUX_URL")
	influxToken := os.Getenv("INFLUX_TOKEN")
	client := influxdb2.NewClient(influxURL, influxToken)
	defer client.Close()

	hc, err := client.Health(context.Background())
	if err != nil {
		log.Fatalf("failed influxDB health check: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"Region": os.Getenv("FLY_REGION"),
			"Status": string(hc.Status),
		}

		t.ExecuteTemplate(w, "index.html.tmpl", data)
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
