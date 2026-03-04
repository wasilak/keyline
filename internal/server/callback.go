package server

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// handleCallback handles the /auth/callback endpoint for OIDC
func (s *Server) handleCallback(c echo.Context) error {
	ctx := c.Request().Context()

	slog.InfoContext(ctx, "OIDC callback received")

	// Check if OIDC is enabled
	if s.oidcProvider == nil {
		slog.ErrorContext(ctx, "OIDC callback received but OIDC is not enabled")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "OIDC is not enabled",
		})
	}

	// Extract query parameters
	stateParam := c.QueryParam("state")
	code := c.QueryParam("code")
	errorParam := c.QueryParam("error")
	errorDesc := c.QueryParam("error_description")

	// Complete the OIDC callback flow
	redirectURL, cookie, err := s.oidcProvider.CompleteCallback(
		ctx,
		s.cache,
		stateParam,
		code,
		errorParam,
		errorDesc,
		s.config.Session.TTL,
	)
	if err != nil {
		slog.ErrorContext(ctx, "OIDC callback failed",
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Authentication failed",
		})
	}

	// Set session cookie
	c.SetCookie(cookie)

	// Redirect to original URL
	slog.InfoContext(ctx, "OIDC callback successful, redirecting",
		slog.String("redirect_url", redirectURL),
	)

	return c.Redirect(http.StatusFound, redirectURL)
}
