package controller

import "github.com/prometheus/client_golang/prometheus"

var (
	clbTotal = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "clb_operator",
				Subsystem: "controller",
				Name:      "clb_total",
				Help:      "Total number of classic loadbalance managed by the controller",
			})

	clbRequestTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "clb_operator",
				Subsystem: "controller",
				Name:      "request_total",
				Help:      "Total request number of the clb",
			},
			[]string{"clb_name", "namespace"},
	)

	clbResponseTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clb_operator",
		Subsystem: "controller",
		Name:      "response_total",
		Help:      "Total response number of the clb",
	})

	clbRequestBytesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clb_operator",
		Subsystem: "controller",
		Name:      "request_bytes_total",
		Help:      "Total request bytes of the clb",
	})

	clbResponseBytesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "clb_operator",
		Subsystem: "controller",
		Name:      "response_bytes_total",
		Help:      "Total response bytes of the clb",
	})
)

func init() {
	prometheus.MustRegister(clbTotal)
	prometheus.MustRegister(clbRequestTotal)
	prometheus.MustRegister(clbResponseTotal)
	prometheus.MustRegister(clbRequestBytesTotal)
	prometheus.MustRegister(clbResponseBytesTotal)
}