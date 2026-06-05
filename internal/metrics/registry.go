package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func NewCounter(name, help string, labels ...string) *prometheus.CounterVec {
	return promauto.NewCounterVec(
		prometheus.CounterOpts{Name: name, Help: help},
		labels,
	)
}

func NewGauge(name, help string, labels ...string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{Name: name, Help: help},
		labels,
	)
}

func NewHistogram(name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
	opts := prometheus.HistogramOpts{
		Name: name,
		Help: help,
	}
	if buckets != nil {
		opts.Buckets = buckets
	}

	return promauto.NewHistogramVec(opts, labels)
}

func NewSimpleCounter(name, help string) prometheus.Counter {
	return promauto.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
}

func NewSimpleGauge(name, help string) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})
}

func NewSimpleHistogram(name, help string, buckets []float64) prometheus.Histogram {
	opts := prometheus.HistogramOpts{
		Name: name,
		Help: help,
	}
	if buckets != nil {
		opts.Buckets = buckets
	}

	return promauto.NewHistogram(opts)
}
