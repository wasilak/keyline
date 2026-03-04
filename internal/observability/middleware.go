package observability

import (
	"net/http"
	"sync/atomic"

	"github.com/labstack/echo/v4"
)

// ConcurrentRequestLimiter creates middleware that limits concurrent requests
func ConcurrentRequestLimiter(maxConcurrent int) echo.MiddlewareFunc {
	var currentRequests int64

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Increment concurrent request counter
			current := atomic.AddInt64(&currentRequests, 1)
			ConcurrentRequests.Set(float64(current))

			// Check if limit exceeded
			if current > int64(maxConcurrent) {
				atomic.AddInt64(&currentRequests, -1)
				ConcurrentRequests.Set(float64(current - 1))
				Errors.WithLabelValues("concurrent_limit_exceeded").Inc()
				return c.JSON(http.StatusServiceUnavailable, map[string]string{
					"error": "Server overloaded",
				})
			}

			// Ensure decrement happens even if handler panics
			defer func() {
				current := atomic.AddInt64(&currentRequests, -1)
				ConcurrentRequests.Set(float64(current))
			}()

			return next(c)
		}
	}
}

// RequestBodySizeLimiter creates middleware that limits request body size
func RequestBodySizeLimiter(maxBytes int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check Content-Length header
			if c.Request().ContentLength > maxBytes {
				Errors.WithLabelValues("request_too_large").Inc()
				return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{
					"error": "Request too large",
				})
			}

			// Wrap request body with LimitReader to enforce limit
			c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxBytes)

			return next(c)
		}
	}
}
