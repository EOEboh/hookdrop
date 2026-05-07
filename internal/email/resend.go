package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Sender struct {
	apiKey   string
	fromAddr string
}

func NewSender(apiKey, fromAddr string) *Sender {
	return &Sender{apiKey: apiKey, fromAddr: fromAddr}
}

type emailPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (s *Sender) SendMagicLink(toEmail, magicURL string) error {
	body := emailPayload{
		From:    s.fromAddr,
		To:      []string{toEmail},
		Subject: "Your hookdrop login link",
		HTML:    buildEmailHTML(magicURL),
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("resend API error: %d", resp.StatusCode)
	}
	return nil
}

func buildEmailHTML(magicURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; background: #09090b; color: #fafafa; padding: 40px;">
  <div style="max-width: 480px; margin: 0 auto;">
    <h1 style="font-size: 24px; margin-bottom: 8px;">⚡ hookdrop</h1>
    <p style="color: #a1a1aa; margin-bottom: 32px;">Click the button below to log in. This link expires in 15 minutes and can only be used once.</p>
    <a href="%s"
       style="display: inline-block; background: #10b981; color: white;
              padding: 12px 24px; border-radius: 8px; text-decoration: none;
              font-weight: 600; font-size: 14px;">
      Log in to hookdrop
    </a>
    <p style="color: #52525b; font-size: 12px; margin-top: 32px;">
      If you didn't request this, you can safely ignore it.
    </p>
  </div>
</body>
</html>`, magicURL)
}
