package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsOnce sync.Once

	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	dbUp                prometheus.GaugeFunc

	dbPingMu sync.RWMutex
	dbPing   func(context.Context) error
)

func setDBPing(f func(context.Context) error) {
	if f == nil {
		return
	}
	dbPingMu.Lock()
	dbPing = f
	dbPingMu.Unlock()
}

func getDBPing() func(context.Context) error {
	dbPingMu.RLock()
	f := dbPing
	dbPingMu.RUnlock()
	return f
}

func initMetrics() {
	metricsOnce.Do(func() {
		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "ispo",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status"},
		)
		httpRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "ispo",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		)

		dbUp = prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "ispo",
				Subsystem: "db",
				Name:      "up",
				Help:      "Whether the database is reachable (1) or not (0).",
			},
			func() float64 {
				ping := getDBPing()
				if ping == nil {
					return 0
				}
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel()
				if err := ping(ctx); err != nil {
					return 0
				}
				return 1
			},
		)

		prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, dbUp)
	})
}

func metricsMiddleware(dbPing func(context.Context) error) gin.HandlerFunc {
	initMetrics()
	setDBPing(dbPing)
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		dur := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path, status).Observe(dur)
	}
}

func metricsHandler(dbPing func(context.Context) error) gin.HandlerFunc {
	initMetrics()
	setDBPing(dbPing)
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func metricsHealthHandler(dbPing func(context.Context) error) gin.HandlerFunc {
	initMetrics()
	setDBPing(dbPing)
	return func(c *gin.Context) {
		ping := getDBPing()
		if ping == nil {
			c.JSON(http.StatusOK, gin.H{"metrics": "ok", "db": "unknown"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
		defer cancel()
		if err := ping(ctx); err != nil {
			writeError(c, http.StatusServiceUnavailable, "service_unavailable", "db", "db down")
			return
		}
		c.JSON(http.StatusOK, gin.H{"metrics": "ok", "db": "ok"})
	}
}
