package core

import (
	"os"
)

var (
	oktaIssuer   = os.Getenv("OKTA_ISSUER_AP")
	oktaRedirect = os.Getenv("OKTA_REDIRECT_URL_AP")
)

type Settings struct {
	TokenURL              string
	RevokeURL             string
	RedirectURI           string
	IntrospectURI         string
	EndSessionURI         string
	PostLogoutRedirectURI string
}

var OktaSettings = Settings{
	TokenURL:              oktaIssuer + "/v1/token",
	RevokeURL:             oktaIssuer + "/v1/revoke",
	RedirectURI:           oktaRedirect + "login",
	IntrospectURI:         oktaIssuer + "/v1/introspect",
	EndSessionURI:         oktaIssuer + "/v1/logout",
	PostLogoutRedirectURI: oktaRedirect + "logout",
}
