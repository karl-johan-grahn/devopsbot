package routes

import (
	"net/http"

	"github.com/karl-johan-grahn/devopsbot/metrics"
)

func HealthHandler(ns string) http.Handler {
	m := http.NewServeMux()
	m.Handle("/metrics", metrics.RegisterPrometheus(ns))
	m.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	m.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return m
}
