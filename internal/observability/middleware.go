package observability

import (
	"net/http"
	"sync/atomic"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

// AuthSpanEnhancer creates middleware that adds authentication attributes to the current span
// This middleware should be placed after authentication has occurred
func AuthSpanEnhancer() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Execute the handler first
			err := next(c)

			// Get the current span from context
			span := trace.SpanFromContext(c.Request().Context())
			if span.IsRecording() {
				// Add authentication attributes if available
				if authMethod := c.Get("auth_method"); authMethod != nil {
					span.SetAttributes(attribute.String("auth.method", authMethod.(string)))
				}
				if authResult := c.Get("auth_result"); authResult != nil {
					span.SetAttributes(attribute.String("auth.result", authResult.(string)))
				}
				if username := c.Get("username"); username != nil {
					span.SetAttributes(attribute.String("auth.username", username.(string)))
				}
			}

			return err
		}
	}
}

// RequestTracingMiddleware creates middleware that ensures proper span attributes
// This complements otelecho middleware by adding custom attributes
func RequestTracingMiddleware() echo.MiddlewareFunc {
	tracer := otel.Tracer("keyline")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// Create a child span for the request (otelecho already creates parent)
			// This allows us to add custom attributes
			ctx, span := tracer.Start(ctx, "keyline.request")
			defer span.End()

			// Update request context
			c.SetRequest(c.Request().WithContext(ctx))

			// Add standard HTTP attributes (otelecho does this too, but we ensure they're present)
			span.SetAttributes(
				attribute.String("http.method", c.Request().Method),
				attribute.String("http.url", c.Request().URL.String()),
				attribute.String("http.target", c.Request().URL.Path),
				attribute.String("http.host", c.Request().Host),
			)

			// Execute handler
			err := next(c)

			// Add response status code
			span.SetAttributes(attribute.Int("http.status_code", c.Response().Status))

			// Add authentication attributes if available
			if authMethod := c.Get("auth_method"); authMethod != nil {
				span.SetAttributes(attribute.String("auth.method", authMethod.(string)))
			}
			if authResult := c.Get("auth_result"); authResult != nil {
				span.SetAttributes(attribute.String("auth.result", authResult.(string)))
			}
			if username := c.Get("username"); username != nil {
				span.SetAttributes(attribute.String("auth.username", username.(string)))
			}

			return err
		}
	}
}
