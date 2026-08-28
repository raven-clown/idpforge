package captcha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const hcaptchaVerifyURL = "https://hcaptcha.com/siteverify"

type hcaptchaVerifier struct {
	secretKey string
}

func (v *hcaptchaVerifier) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	form := url.Values{
		"secret":   {v.secretKey},
		"response": {token},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hcaptchaVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Success, nil
}
