package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// HashSessionID creates a SHA256 hash of a session ID for safe logging
// This prevents full session IDs from appearing in logs while still allowing correlation
func HashSessionID(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for brevity
}

// ExtractSourceIP extracts the source IP from the request
// Checks X-Forwarded-For, X-Real-IP, and RemoteAddr in that order
func ExtractSourceIP(req *http.Request) string {
	// Check X-Forwarded-For header (may contain multiple IPs)
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in format "IP:port", so we need to strip the port
	addr := req.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}

	return addr
}
