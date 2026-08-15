package auth

// Cookie names for browser sessions. Values are JWTs; flags are set by the HTTP layer.
const (
	AccessCookieName  = "unicore_access"
	RefreshCookieName = "unicore_refresh"
	// RefreshCookiePath limits the refresh JWT to auth routes so it is not
	// attached to every API call.
	RefreshCookiePath = "/api/v1/auth"
	AuthModeHeader    = "X-Auth-Mode"
	AuthModeBearer    = "bearer"
)
