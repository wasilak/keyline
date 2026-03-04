package server

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/yourusername/keyline/internal/session"
)

// handleLogout handles the /auth/logout endpoint
func (s *Server) handleLogout(c echo.Context) error {
	ctx := c.Request().Context()

	slog.InfoContext(ctx, "Logout request received")

	// Extract session cookie
	sessionCookie, err := c.Cookie(s.config.Session.CookieName)
	if err != nil || sessionCookie == nil {
		// No session cookie found
		slog.InfoContext(ctx, "No active session to logout")
		return c.JSON(http.StatusOK, map[string]string{
			"message": "No active session",
		})
	}

	sessionID := sessionCookie.Value
	if sessionID != "" {
		// Delete session from store
		if err := session.DeleteSession(ctx, s.cache, sessionID); err != nil {
			slog.ErrorContext(ctx, "Failed to delete session",
				slog.String("error", err.Error()),
			)
			// Continue anyway to clear the cookie
		} else {
			slog.InfoContext(ctx, "Session deleted",
				slog.String("session_id", sessionID),
			)
		}
	}

	// Clear session cookie
	clearCookie := &http.Cookie{
		Name:     s.config.Session.CookieName,
		Value:    "",
		Path:     s.config.Session.CookiePath,
		Domain:   s.config.Session.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(clearCookie)

	// Check if OIDC is enabled and has end_session_endpoint
	if s.oidcProvider != nil {
		doc := s.oidcProvider.GetDiscoveryDoc()
		if doc != nil && doc.EndSessionEndpoint != "" {
			// Redirect to OIDC provider logout
			slog.InfoContext(ctx, "Redirecting to OIDC provider logout",
				slog.String("end_session_endpoint", doc.EndSessionEndpoint),
			)
			return c.Redirect(http.StatusFound, doc.EndSessionEndpoint)
		}
	}

	// No OIDC logout endpoint, return success
	slog.InfoContext(ctx, "Logout successful")
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out",
	})
}
