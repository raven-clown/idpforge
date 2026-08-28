package captcha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

var recaptchaVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

type recaptchaVerifier struct {
	secretKey string
}

func (v *recaptchaVerifier) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	form := url.Values{
		"secret":   {v.secretKey},
		"response": {token},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, recaptchaVerifyURL, nil)
	if err != nil {
		return false, err
	}
	req.URL.RawQuery = form.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool     `json:"success"`
		Score   float64  `json:"score"`
		Errors  []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	// v3 returns a risk score instead of a plain pass/fail; treat below 0.5
	// as failed, matching Google's documented default threshold.
	if result.Score > 0 && result.Score < 0.5 {
		return false, nil
	}
	return result.Success, nil
}
