package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
)

type Metrics struct {
	Registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func New(hub *realtime.Hub) *Metrics {
	m := &Metrics{
		Registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "messenger_http_requests_total", Help: "Completed HTTP requests."}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "messenger_http_duration_seconds", Help: "HTTP handler duration, excluding WebSocket sessions.", Buckets: prometheus.DefBuckets}, []string{"method", "route"}),
	}
	m.Registry.MustRegister(m.requests, m.duration, collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "messenger_websocket_connections", Help: "Active local WebSocket connections."}, hub.TotalConnections),
		prometheus.NewCounterFunc(prometheus.CounterOpts{Name: "messenger_websocket_slow_clients_total", Help: "Slow WebSocket clients disconnected."}, hub.SlowClients),
		prometheus.NewCounterFunc(prometheus.CounterOpts{Name: "messenger_redis_publish_errors_total", Help: "Failed cross-instance event publications."}, hub.PublishFailures),
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			route := "unmatched"
			if ctx := chi.RouteContext(r.Context()); ctx != nil && ctx.RoutePattern() != "" {
				route = ctx.RoutePattern()
			}
			method := r.Method
			switch method {
			case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
			default:
				method = "OTHER"
			}
			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}
			m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
			if status != http.StatusSwitchingProtocols {
				m.duration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
			}
		}()
		next.ServeHTTP(wrapped, r)
	})
}
