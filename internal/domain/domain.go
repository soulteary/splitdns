// Package domain provides DNS domain validation, normalization and
// longest-suffix matching used to map hostnames to resolver files.
package domain

import (
	"fmt"
	"strings"
)

// Warning describes a non-fatal issue detected during normalization.
type Warning struct {
	Message string
}

// Result is the outcome of normalizing a user-supplied domain suffix.
type Result struct {
	// Normalized is the cleaned, lowercase domain without a trailing dot.
	Normalized string
	// Labels are the individual DNS labels of the normalized domain.
	Labels []string
	// Warnings collects non-fatal advisories (e.g. single-label, .local).
	Warnings []Warning
}

const maxDomainLength = 253
const maxLabelLength = 63

// Normalize validates and normalizes a domain suffix. The allowSingleLabel and
// allowLocal flags relax specific safety checks (typically enabled via --force).
// Warnings are always populated even when the corresponding check is allowed.
func Normalize(input string, allowSingleLabel, allowLocal bool) (Result, error) {
	var res Result

	if input == "" {
		return res, fmt.Errorf("domain must not be empty")
	}

	if strings.ContainsRune(input, 0) {
		return res, fmt.Errorf("domain must not contain NUL characters")
	}

	for _, r := range input {
		if r < 0x20 || r == 0x7f {
			return res, fmt.Errorf("domain must not contain control characters")
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return res, fmt.Errorf("domain must not contain whitespace")
		}
	}

	if strings.ContainsAny(input, "/\\") {
		return res, fmt.Errorf("domain must not contain path separators (/ or \\)")
	}

	lowered := strings.ToLower(strings.TrimSpace(input))
	lowered = strings.TrimSuffix(lowered, ".")

	if lowered == "" {
		return res, fmt.Errorf("domain must not be empty after normalization")
	}

	if strings.HasPrefix(lowered, ".") {
		return res, fmt.Errorf("domain must not start with a dot")
	}
	if strings.Contains(lowered, "..") {
		return res, fmt.Errorf("domain must not contain empty labels or path traversal (..)")
	}

	if len(lowered) > maxDomainLength {
		return res, fmt.Errorf("domain exceeds maximum length of %d characters", maxDomainLength)
	}

	labels := strings.Split(lowered, ".")
	for _, label := range labels {
		if err := validateLabel(label); err != nil {
			return res, err
		}
	}

	res.Normalized = lowered
	res.Labels = labels

	if len(labels) == 1 {
		if !allowSingleLabel {
			return res, fmt.Errorf("single-label domain %q is unusual; use --force to allow it", lowered)
		}
		res.Warnings = append(res.Warnings, Warning{
			Message: fmt.Sprintf("single-label domain %q may match broadly; ensure this is intended", lowered),
		})
	}

	if isLocalDomain(labels) {
		w := Warning{
			Message: fmt.Sprintf("%q uses the reserved .local suffix used by mDNS/Bonjour; a resolver override may interfere with local discovery", lowered),
		}
		res.Warnings = append(res.Warnings, w)
		if !allowLocal {
			return res, fmt.Errorf("%q; use --force to proceed", w.Message)
		}
	}

	return res, nil
}

func validateLabel(label string) error {
	if label == "" {
		return fmt.Errorf("domain must not contain empty labels")
	}
	if len(label) > maxLabelLength {
		return fmt.Errorf("label %q exceeds maximum length of %d characters", label, maxLabelLength)
	}
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return fmt.Errorf("label %q must not start or end with a hyphen", label)
	}
	for _, r := range label {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return fmt.Errorf("label %q contains invalid character %q; only letters, digits and hyphens are allowed", label, r)
		}
	}
	return nil
}

func isLocalDomain(labels []string) bool {
	return len(labels) > 0 && labels[len(labels)-1] == "local"
}

// FileName returns the resolver filename for a normalized domain. Since the
// domain is already validated, the name equals the normalized domain.
func FileName(normalized string) string {
	return normalized
}
