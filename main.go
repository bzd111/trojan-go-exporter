package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var opts struct {
	Listen                 string       `short:"l" long:"listen" description:"Listen address" value-name:"[ADDR]:PORT" default:":9550"`
	MetricsPath            string       `short:"m" long:"metrics-path" description:"Metrics path" value-name:"PATH" default:"/scrape"`
	V2RayEndpoint          string       `short:"e" long:"trojan-go-endpoint" description:"Trojan-Go API endpoint" value-name:"HOST:PORT" default:"127.0.0.1:10000"`
	ScrapeTimeoutInSeconds int64        `short:"t" long:"scrape-timeout" description:"The timeout in seconds for every individual scrape" value-name:"N" default:"3"`
	Version                bool         `long:"version" description:"Show version"`
	Call                   func(string) `short:"c" description:"Call phone number"`
}

var exporter *Exporter

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	promhttp.HandlerFor(
		exporter.registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError},
	).ServeHTTP(w, r)
}

func main() {
	opts.Call = func(num string) {
		cmd := exec.Command("open", "callto:"+num)
		cmd.Start()
		cmd.Process.Release()
	}

	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(0)
	}
	scrapeTimeout := time.Duration(opts.ScrapeTimeoutInSeconds) * time.Second
	exporter = NewExporter(opts.V2RayEndpoint, scrapeTimeout)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/scrape", scrapeHandler)

	log.Fatal(http.ListenAndServe(opts.Listen, nil))
}
