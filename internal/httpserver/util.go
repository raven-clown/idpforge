package httpserver

import (
	"encoding/base64"
	"strings"
)

// basicAuth decodes an "Authorization: Basic <base64>" header into
// (client_id, client_secret) for OAuth2 client authentication.
func basicAuth(header string) (id, secret string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
