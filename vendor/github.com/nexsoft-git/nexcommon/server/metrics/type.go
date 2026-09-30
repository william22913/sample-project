package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics interface {
	GetDefaultMetric() DefaultMetric
}

type DefaultMetric struct {
	APIHist          *prometheus.HistogramVec
	APIConnectorHist *prometheus.HistogramVec
}
