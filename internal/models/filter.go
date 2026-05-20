package models

import (
	"net/url"
	"strings"
	"time"
)

type RequestFilter struct {
	Search   string    // free text: matched against body
	Method   string    // GET, POST, PUT, PATCH, DELETE
	Verified string    // verified, failed, unverified
	Since    time.Time // requests after this time
	Limit    int
}

// parse filter params from a URL query string
func FilterFromQuery(q url.Values) RequestFilter {
	f := RequestFilter{
		Search:   strings.TrimSpace(q.Get("search")),
		Method:   strings.ToUpper(strings.TrimSpace(q.Get("method"))),
		Verified: strings.ToLower(strings.TrimSpace(q.Get("verified"))),
		Limit:    100,
	}

	// Date range: translate named ranges to a Since time
	switch q.Get("range") {
	case "1h":
		f.Since = time.Now().UTC().Add(-1 * time.Hour)
	case "24h":
		f.Since = time.Now().UTC().Add(-24 * time.Hour)
	case "7d":
		f.Since = time.Now().UTC().Add(-7 * 24 * time.Hour)
	}

	// Validate method
	valid := map[string]bool{
		"GET": true, "POST": true, "PUT": true,
		"PATCH": true, "DELETE": true, "": true,
	}
	if !valid[f.Method] {
		f.Method = ""
	}

	// Validate verified
	validV := map[string]bool{
		"verified": true, "failed": true, "unverified": true, "": true,
	}
	if !validV[f.Verified] {
		f.Verified = ""
	}

	return f
}
