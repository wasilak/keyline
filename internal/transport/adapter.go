package transport

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Adapter defines the interface for deployment mode adapters
type Adapter interface {
	// HandleRequest processes an incoming request
	HandleRequest(c echo.Context) error

	// Name returns the adapter name
	Name() string
}

// RequestContext contains normalized request information
type RequestContext struct {
	Method      string            // HTTP method
	Path        string            // Request path
	Host        string            // Request host
	Headers     map[string]string // Request headers
	Cookies     []*http.Cookie    // Request cookies
	OriginalURL string            // Full original URL
}
