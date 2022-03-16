package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	_ = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "signaling_active_sessions",
		Help: "The total number of active sessions",
	}, func() float64 {
		return float64(len(sessions))
	})

	_ = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "signaling_active_connections",
		Help: "The total number of active connections",
	}, func() float64 {
		var cnt = 0
		for _, s := range sessions {
			cnt += len(s.Connections)
		}
		return float64(cnt)
	})

	metricSessionsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "signaling_sessions",
		Help: "The total number of created sessions",
	})

	metricConnectionsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "signaling_connections",
		Help: "The total number of created connections",
	})

	metricMessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "signaling_candidates",
		Help: "The total number of messages exchanged",
	}, []string{"type"})

	metricHttpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Count of all HTTP requests",
	}, []string{"code", "method"})

	metricHttpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of all HTTP requests",
	}, []string{"code", "method"})
)
