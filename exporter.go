package main

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/p4gefau1t/trojan-go/api/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"google.golang.org/grpc"
)

// Exporter struct
type Exporter struct {
	sync.Mutex
	endpoint           string
	scrapeTimeout      time.Duration
	registry           *prometheus.Registry
	totalScrapes       prometheus.Counter
	metricDescriptions map[string]*prometheus.Desc
}

// NewExporter to create exporter
func NewExporter(endpoint string, scrapeTimeout time.Duration) *Exporter {
	e := Exporter{
		endpoint:      endpoint,
		scrapeTimeout: scrapeTimeout,
		registry:      prometheus.NewRegistry(),

		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "trojan_go",
			Name:      "scrapes_total",
			Help:      "Total number of scrapes performed",
		}),
	}

	e.metricDescriptions = map[string]*prometheus.Desc{}

	for k, desc := range map[string]struct {
		txt  string
		lbls []string
	}{
		"scrape_duration_seconds":      {txt: "Scrape duration in seconds"},
		"upload_traffic_bytes_total":   {txt: "Number of transmitted bytes", lbls: []string{"target"}},
		"download_traffic_bytes_total": {txt: "Number of receieved bytes", lbls: []string{"target"}},
		"current_upload_speed":         {txt: "Number of current upload speed bytes", lbls: []string{"target"}},
		"current_download_speed":       {txt: "Number of current download speed bytes", lbls: []string{"target"}},
	} {
		e.metricDescriptions[k] = e.newMetricDescr(k, desc.txt, desc.lbls)
	}

	e.registry.MustRegister(&e)

	return &e
}

// Collect metrics
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.Lock()
	defer e.Unlock()
	e.totalScrapes.Inc()

	if err := e.scrapeTrojanGo(ch); err != nil {
		log.Warnf("Scrape failed! %s", err)
	}
	ch <- e.totalScrapes
}

// Describe implement
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range e.metricDescriptions {
		ch <- desc
	}
	ch <- e.totalScrapes.Desc()
}

// scrape TrojanGo metrics
func (e *Exporter) scrapeTrojanGo(ch chan<- prometheus.Metric) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.scrapeTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, e.endpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to dial: %w, timeout: %v", err, e.scrapeTimeout)
	}
	defer conn.Close()

	client := service.NewTrojanServerServiceClient(conn)

	if err := e.scrapeTrojanGoMetrics(ctx, ch, client); err != nil {
		return err
	}
	return nil
}

// scrape TrojanGo Metrics
func (e *Exporter) scrapeTrojanGoMetrics(ctx context.Context, ch chan<- prometheus.Metric, client service.TrojanServerServiceClient) error {
	stream, _ := client.ListUsers(ctx, &service.ListUsersRequest{})
	result := []*service.ListUsersResponse{}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		result = append(result, resp)
	}

	for _, resp := range result {
		userHash := resp.Status.User.Hash
		uploadTraffic := resp.Status.TrafficTotal.UploadTraffic
		e.registerConstMetricGauge(ch, "upload_traffic_bytes_total", float64(uploadTraffic), userHash)
		downloadTraffic := resp.Status.TrafficTotal.DownloadTraffic
		e.registerConstMetricGauge(ch, "download_traffic_bytes_total", float64(downloadTraffic), userHash)
		speedCureent := resp.Status.SpeedCurrent
		if speedCureent != nil {
			uploadSpeed := speedCureent.UploadSpeed
			e.registerConstMetricGauge(ch, "current_upload_speed", float64(uploadSpeed), userHash)
			downloadSpeed := speedCureent.DownloadSpeed
			e.registerConstMetricGauge(ch, "current_download_speed", float64(downloadSpeed), userHash)
		} else {
			e.registerConstMetricGauge(ch, "current_upload_speed", 0, userHash)
			e.registerConstMetricGauge(ch, "current_download_speed", 0, userHash)
		}
	}
	return nil
}

// register Const Metric Gauge
func (e *Exporter) registerConstMetricGauge(ch chan<- prometheus.Metric, metric string, val float64, labels ...string) {
	e.registerConstMetric(ch, metric, val, prometheus.GaugeValue, labels...)
}

// register Const Metric Counter
func (e *Exporter) registerConstMetricCounter(ch chan<- prometheus.Metric, metric string, val float64, labels ...string) {
	e.registerConstMetric(ch, metric, val, prometheus.CounterValue, labels...)
}

// register Const Metric
func (e *Exporter) registerConstMetric(ch chan<- prometheus.Metric, metric string, val float64, valType prometheus.ValueType, labelValues ...string) {
	descr := e.metricDescriptions[metric]
	if descr == nil {
		descr = e.newMetricDescr(metric, metric+" metric", nil)
	}

	if m, err := prometheus.NewConstMetric(descr, valType, val, labelValues...); err == nil {
		ch <- m
	} else {
		log.Debugf("NewConstMetric() err: %s", err)
	}
}

func (e *Exporter) newMetricDescr(metricName string, docString string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName("trojan_go", "", metricName), docString, labels, nil)
}
