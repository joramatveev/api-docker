package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const promeHost string = ""
const promePort string = "2112"

var (
	reqBuckets      = []float64{.001, .02, .035, .06, .1, .35, .75, 1.2, 3.5, 5.5, 10}
	registeredUsers = promauto.NewCounter(prometheus.CounterOpts{
		Name: "number_of_registered_users",
		Help: "All registered users",
	})
	cakesGiven = promauto.NewCounter(prometheus.CounterOpts{
		Name: "number_of_cakes_given",
		Help: "Num of given cakes",
	})
	requestRecords = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_request_record_seconds",
		Help:    "Histogram of response time for handler in seconds.",
		Buckets: reqBuckets,
	}, []string{"path"})
)

func initPrometheus() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe( promeHost+":"+promePort, nil)
}
