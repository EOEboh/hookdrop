package endpoint

import (
	"fmt"
	"regexp"
	"strings"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ValidateSlug(slug string) error {
	slug = strings.TrimSpace(slug)

	if len(slug) < 3 {
		return fmt.Errorf("slug must be at least 3 characters")
	}
	if len(slug) > 48 {
		return fmt.Errorf("slug must be 48 characters or fewer")
	}
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("slug may only contain lowercase letters, numbers, and hyphens")
	}

	// Reserve common paths that would conflict with the API
	reserved := map[string]bool{
		"api": true, "auth": true, "sessions": true,
		"requests": true, "replay": true, "events": true,
		"admin": true, "health": true,
	}
	if reserved[slug] {
		return fmt.Errorf("'%s' is a reserved name", slug)
	}

	return nil
}
