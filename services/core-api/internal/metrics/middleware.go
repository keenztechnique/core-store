package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func PrometheusMiddleware(m *Metrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Обертка для перехвата HTTP-статуса ответа
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(ww.Status())
			path := r.URL.Path
			method := r.Method

			// Увеличиваем счетчик запросов
			m.HttpRequestsTotal.WithLabelValues(method, path, status).Inc()

			// Записываем гистограмму длительности
			m.HttpRequestDuration.WithLabelValues(method, path).Observe(duration)
		})
	}
}
