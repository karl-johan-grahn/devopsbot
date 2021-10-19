package metrics

import (
	"net/http"
	"runtime"

	"github.com/karl-johan-grahn/devopsbot/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	MetricsRegisterer = prometheus.DefaultRegisterer
	MetricsGatherer   = prometheus.DefaultGatherer
)

func RegisterPrometheus(namespace string) http.Handler {
	buildInfo(namespace, version.Version, version.Revision)
	return promhttp.HandlerFor(MetricsGatherer, promhttp.HandlerOpts{})
}

// Creates, sets, and registers a gauge metric for build information
func buildInfo(namespace, version, revision string) {
	m := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "build_info",
			Help:      "A constant metric labeled with build information for " + namespace,
		},
		[]string{"version", "revision", "goversion", "goarch", "goos"},
	)
	MetricsRegisterer.MustRegister(m)
	m.WithLabelValues(version, revision, runtime.Version(), runtime.GOARCH, runtime.GOOS).Set(1)
}
