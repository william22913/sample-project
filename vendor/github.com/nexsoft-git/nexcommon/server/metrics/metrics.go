package metrics

import "github.com/prometheus/client_golang/prometheus"

type metrics struct {
	def DefaultMetric
}

func NewMetrics(
	cols ...prometheus.Collector,
) Metrics {
	var metric metrics

	metric.def.APIHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "my_app_http_hist",
			Help: "Histogram of requests.",
		},
		[]string{"path", "method", "status"},
	)

	metric.def.APIConnectorHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_connector_hist",
			Help: "Histogram of requests to another resource",
		},
		[]string{"host", "path", "method", "status"},
	)

	prometheus.Register(metric.def.APIHist)
	prometheus.Register(metric.def.APIConnectorHist)

	for i := range cols {
		prometheus.Register(cols[i])
	}

	return &metric
}

func (m *metrics) GetDefaultMetric() DefaultMetric {
	return m.def
}
