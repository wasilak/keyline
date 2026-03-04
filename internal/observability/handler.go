package observability

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler returns an Echo handler for the Prometheus metrics endpoint
func MetricsHandler() echo.HandlerFunc {
	handler := promhttp.Handler()
	return func(c echo.Context) error {
		handler.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
