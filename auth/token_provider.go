package auth

// TokenProvider is the contract for anything that can supply a bearer token.
// The running agent calls GetAccessToken() before every cloud request.
// Implementations: StaticKeyProvider (now), OAuthProvider (Phase 3b).
type TokenProvider interface {
	GetAccessToken() (string, error)
}
