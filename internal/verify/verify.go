package verify

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/models"
)

type Result struct {
	Status   string // "verified", "failed"
	Provider string
	Reason   string
}

// Verify attempts to verify a request against all stored secrets for an endpoint.
// It returns the first successful verification, or the last failure if none succeed.
func Verify(req *models.CapturedRequest, secrets []*models.WebhookSecret) Result {
	if len(secrets) == 0 {
		return Result{Status: "unverified", Provider: "", Reason: "no secret configured"}
	}

	var lastFailure Result

	for _, secret := range secrets {
		result := verifyWithProvider(req, secret)
		if result.Status == "verified" {
			return result
		}
		lastFailure = result
	}

	return lastFailure
}

func verifyWithProvider(req *models.CapturedRequest, secret *models.WebhookSecret) Result {
	switch secret.Provider {
	case "stripe":
		return verifyStripe(req, secret.Secret)
	case "paystack":
		return verifyPaystack(req, secret.Secret)
	case "github":
		return verifyGitHub(req, secret.Secret)
	default:
		return verifyGeneric(req, secret.Secret, secret.Provider)
	}
}

// verifyStripe handles Stripe's timestamp+body signature scheme
// Stripe-Signature: t=1712345678,v1=abc123...
func verifyStripe(req *models.CapturedRequest, secret string) Result {
	provider := "stripe"

	sigHeader := req.Headers["Stripe-Signature"]
	if sigHeader == "" {
		return Result{Status: "failed", Provider: provider, Reason: "Stripe-Signature header missing"}
	}

	// Parse the header into parts
	parts := make(map[string]string)
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			parts[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	timestamp := parts["t"]
	signature := parts["v1"]

	if timestamp == "" || signature == "" {
		return Result{Status: "failed", Provider: provider, Reason: "malformed Stripe-Signature header"}
	}

	// Reject timestamps older than 5 minutes — replay attack protection
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return Result{Status: "failed", Provider: provider, Reason: "invalid timestamp"}
	}
	if time.Now().Unix()-ts > 300 {
		return Result{
			Status:   "failed",
			Provider: provider,
			Reason:   fmt.Sprintf("timestamp too old (%ds) — possible replay attack", time.Now().Unix()-ts),
		}
	}

	// Stripe signed payload = "{timestamp}.{raw_body}"
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(req.Body))
	expected := computeHMAC(sha256.New, signedPayload, secret)

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return Result{Status: "failed", Provider: provider, Reason: "signature mismatch"}
	}

	return Result{Status: "verified", Provider: provider, Reason: "HMAC-SHA256 verified"}
}

// verifyPaystack handles Paystack's simple HMAC-SHA512
// X-Paystack-Signature: abc123...
func verifyPaystack(req *models.CapturedRequest, secret string) Result {
	provider := "paystack"

	sigHeader := req.Headers["X-Paystack-Signature"]
	if sigHeader == "" {
		return Result{Status: "failed", Provider: provider, Reason: "X-Paystack-Signature header missing"}
	}

	expected := computeHMAC(sha512.New, string(req.Body), secret)

	if !hmac.Equal([]byte(expected), []byte(sigHeader)) {
		return Result{Status: "failed", Provider: provider, Reason: "signature mismatch"}
	}

	return Result{Status: "verified", Provider: provider, Reason: "HMAC-SHA512 verified"}
}

// verifyGitHub handles GitHub's sha256= prefixed signature
// X-Hub-Signature-256: sha256=abc123...
func verifyGitHub(req *models.CapturedRequest, secret string) Result {
	provider := "github"

	sigHeader := req.Headers["X-Hub-Signature-256"]
	if sigHeader == "" {
		return Result{Status: "failed", Provider: provider, Reason: "X-Hub-Signature-256 header missing"}
	}

	if !strings.HasPrefix(sigHeader, "sha256=") {
		return Result{Status: "failed", Provider: provider, Reason: "unexpected signature format"}
	}

	signature := strings.TrimPrefix(sigHeader, "sha256=")
	expected := computeHMAC(sha256.New, string(req.Body), secret)

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return Result{Status: "failed", Provider: provider, Reason: "signature mismatch"}
	}

	return Result{Status: "verified", Provider: provider, Reason: "HMAC-SHA256 verified"}
}

// verifyGeneric handles any provider using plain HMAC-SHA256
func verifyGeneric(req *models.CapturedRequest, secret, provider string) Result {
	// Try common generic signature headers in order
	candidates := []string{
		"X-Signature-256",
		"X-Webhook-Signature",
		"X-Signature",
		"Webhook-Signature",
	}

	var sigHeader string
	var sigHeaderName string
	for _, name := range candidates {
		if val := req.Headers[name]; val != "" {
			sigHeader = val
			sigHeaderName = name
			break
		}
	}

	if sigHeader == "" {
		return Result{
			Status:   "failed",
			Provider: provider,
			Reason:   "no recognised signature header found",
		}
	}

	// Strip common prefixes like "sha256=" or "hmac="
	sig := sigHeader
	for _, prefix := range []string{"sha256=", "hmac=", "v1="} {
		sig = strings.TrimPrefix(sig, prefix)
	}

	expected := computeHMAC(sha256.New, string(req.Body), secret)

	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return Result{
			Status:   "failed",
			Provider: provider,
			Reason:   fmt.Sprintf("HMAC-SHA256 mismatch on %s", sigHeaderName),
		}
	}

	return Result{Status: "verified", Provider: provider, Reason: fmt.Sprintf("HMAC-SHA256 verified via %s", sigHeaderName)}
}

// computeHMAC computes a lowercase hex HMAC using the given hash constructor
func computeHMAC(h func() hash.Hash, payload, secret string) string {
	mac := hmac.New(h, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
