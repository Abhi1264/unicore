package services

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Branding is untrusted institute-admin input (XSS/CSS injection risk).
// Unknown keys are refused so the store never holds unvalidated fields.

const (
	maxBrandingBytes     = 4 << 10
	maxGradingScaleBytes = 16 << 10
	maxDisplayNameRunes  = 200
	maxBrandingURLLength = 2048
)

var (
	validGradingSystems = map[string]struct{}{
		"cgpa": {}, "percentage": {}, "letter": {},
	}
	validCalendarTypes = map[string]struct{}{
		"semester": {}, "annual": {},
	}
)

type brandingConfig struct {
	PrimaryColor         *string `json:"primary_color,omitempty"`
	SecondaryColor       *string `json:"secondary_color,omitempty"`
	LogoURL              *string `json:"logo_url,omitempty"`
	FaviconURL           *string `json:"favicon_url,omitempty"`
	InstituteDisplayName *string `json:"institute_display_name,omitempty"`
}

// validateBranding parses, checks, and re-serialises so unknown keys are dropped.
func validateBranding(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) > maxBrandingBytes {
		return nil, fmt.Errorf("%w: branding must be under %d bytes", ErrInvalidInput, maxBrandingBytes)
	}

	// Bare null decodes to a zero struct; clear branding with `{}` instead.
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var cfg brandingConfig
	if strings.TrimSpace(string(raw)) == "null" || dec.Decode(&cfg) != nil {
		return nil, fmt.Errorf("%w: branding must be an object with only the supported keys (primary_color, secondary_color, logo_url, favicon_url, institute_display_name)", ErrInvalidInput)
	}

	for field, v := range map[string]*string{
		"primary_color":   cfg.PrimaryColor,
		"secondary_color": cfg.SecondaryColor,
	} {
		if v == nil {
			continue
		}
		if !isHexColor(*v) {
			return nil, fmt.Errorf("%w: %s must be a hex colour such as #1a73e8", ErrInvalidInput, field)
		}
	}

	for field, v := range map[string]**string{
		"logo_url":    &cfg.LogoURL,
		"favicon_url": &cfg.FaviconURL,
	} {
		if *v == nil {
			continue
		}
		if strings.TrimSpace(**v) == "" {
			*v = nil
			continue
		}
		if err := validateImageURL(field, **v); err != nil {
			return nil, err
		}
	}

	if cfg.InstituteDisplayName != nil {
		name := strings.TrimSpace(*cfg.InstituteDisplayName)
		if err := validateName("institute_display_name", name); err != nil {
			return nil, err
		}
		cfg.InstituteDisplayName = &name
	}

	out, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: branding could not be encoded", ErrInvalidInput)
	}
	return out, nil
}

func isHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// validateImageURL allows only absolute http(s) URLs (rejects javascript:/data:).
func validateImageURL(field, raw string) error {
	if len(raw) > maxBrandingURLLength {
		return fmt.Errorf("%w: %s is too long", ErrInvalidInput, field)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: %s must be a valid URL", ErrInvalidInput, field)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: %s must be an http(s) URL", ErrInvalidInput, field)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: %s must be an absolute URL", ErrInvalidInput, field)
	}
	return nil
}

func validateGradingScale(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) > maxGradingScaleBytes {
		return nil, fmt.Errorf("%w: grading_scale must be under %d bytes", ErrInvalidInput, maxGradingScaleBytes)
	}
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("%w: grading_scale must be a JSON object", ErrInvalidInput)
	}
	return raw, nil
}

func validateEnum(field, value string, allowed map[string]struct{}) error {
	if _, ok := allowed[value]; !ok {
		return fmt.Errorf("%w: %s must be one of %s", ErrInvalidInput, field, sortedKeys(allowed))
	}
	return nil
}

func sortedKeys(m map[string]struct{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return strings.Join(keys, ", ")
}
