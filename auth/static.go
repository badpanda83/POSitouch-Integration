package auth

// StaticKeyProvider wraps the existing api_key string so it satisfies
// TokenProvider. Drop-in replacement for the current cfg.Cloud.APIKey usage.
// Replace this with OAuthProvider when the IdP is chosen.
type StaticKeyProvider struct {
	Key string
}

func (s *StaticKeyProvider) GetAccessToken() (string, error) {
	return s.Key, nil
}
