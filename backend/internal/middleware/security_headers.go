package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders adds standard defensive HTTP response headers to every reply.
//
// Header rationale:
//   - X-Content-Type-Options: nosniff       — prevent MIME-type sniffing attacks
//   - X-Frame-Options: DENY                 — block clickjacking via iframes
//   - X-XSS-Protection: 1; mode=block        — enable legacy XSS filter as defense-in-depth
//   - Referrer-Policy: strict-origin-when-cross-origin — limit referrer leakage
//   - Permissions-Policy                    — deny camera/mic/geolocation access
//   - Strict-Transport-Security             — HSTS, only sent over TLS connections
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// HSTS is only meaningful over an encrypted connection; skip it over plain HTTP
		// so localhost development doesn't get a sticky HSTS entry in the browser.
		// Also check X-Forwarded-Proto for deployments behind a TLS-terminating reverse proxy.
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
