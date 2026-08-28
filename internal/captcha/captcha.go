// Package captcha verifies a challenge token against the configured
// provider. Set IDPFORGE_CAPTCHA_PROVIDER=none to skip verification (default).
package captcha

import "context"

type Verifier interface {
	Verify(ctx context.Context, token, remoteIP string) (bool, error)
}

type noopVerifier struct{}

func (noopVerifier) Verify(context.Context, string, string) (bool, error) { return true, nil }

func New(provider, secretKey string) Verifier {
	switch provider {
	case "turnstile":
		return &turnstileVerifier{secretKey: secretKey}
	case "hcaptcha":
		return &hcaptchaVerifier{secretKey: secretKey}
	case "recaptcha":
		return &recaptchaVerifier{secretKey: secretKey}
	default:
		return noopVerifier{}
	}
}
