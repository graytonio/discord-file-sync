package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	CommandsServed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "discord_git_sync",
		Name: "commands_total",
	}, []string{"command"})

	CommandsFailed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "discord_git_sync",
		Name: "commands_failed",
	}, []string{"command"})
)

func StartMetrics() {
	prometheus.MustRegister(CommandsServed)
	prometheus.MustRegister(CommandsFailed)

	http.Handle("/metrics", promhttp.Handler())
	logrus.WithError(http.ListenAndServe(":2112", nil)).Fatal("metrics server crashed")
}