package main

import "fmt"

// activate is the OAuth activation gate. It will open a browser for the IT
// person to authenticate with the company IdP, exchange the authorization code
// for tokens, and write them securely to disk.
//
// TODO(phase-3b): implement once IdP provider is chosen (Okta, Azure AD, Auth0, etc.)
// See auth/oauth.go for the token provider stub.
func runActivate() error {
	fmt.Println()
	fmt.Println("  OAuth activation is not yet available.")
	fmt.Println("  The auth.OAuthProvider will be implemented in Phase 3b once")
	fmt.Println("  your company's Identity Provider (IdP) has been selected.")
	fmt.Println()
	fmt.Println("  For now, set 'api_key' in rooam_config.json directly.")
	fmt.Println()
	return nil
}
