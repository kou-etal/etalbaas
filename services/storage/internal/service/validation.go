package service

import (
	"fmt"
	"regexp"
	"strings"
)

var bucketNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

var validAccessLevels = map[string]bool{
	"public":    true,
	"protected": true,
	"private":   true,
}

func validateBucketName(name string) error {
	if !bucketNameRe.MatchString(name) {
		return fmt.Errorf("bucket name must be 3-63 characters, lowercase alphanumeric and hyphens, starting and ending with alphanumeric")
	}
	if strings.Contains(name, "--") {
		return fmt.Errorf("bucket name must not contain consecutive hyphens")
	}
	return nil
}

func validateAccessLevel(level string) error {
	if !validAccessLevels[level] {
		return fmt.Errorf("access_level must be one of: public, protected, private")
	}
	return nil
}

func validateFileSizeLimit(limit int64) error {
	if limit < 0 {
		return fmt.Errorf("file_size_limit must be non-negative")
	}
	return nil
}

func validateMIMETypes(types []string) error {
	for _, t := range types {
		if t == "" {
			return fmt.Errorf("allowed_mime_types must not contain empty strings")
		}
		if !strings.Contains(t, "/") {
			return fmt.Errorf("invalid MIME type %q: must contain '/'", t)
		}
	}
	return nil
}
