package auth

import "errors"

// OAuthProvider will implement the Client Credentials flow against an
// OIDC-compliant IdP (Okta, Azure AD, Auth0, etc.).
// Configured via config.OAuthConfig once an IdP is chosen.
//
// TODO(phase-3b): implement token fetch, caching, and background refresh.
type OAuthProvider struct {
	// TODO(phase-3b): add fields: providerURL, clientID, clientSecret, scopes, httpClient, cachedToken, tokenExpiry
}

func (o *OAuthProvider) GetAccessToken() (string, error) {
	return "", errors.New("OAuthProvider not yet implemented — use StaticKeyProvider")
}
