package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateBrandingAcceptsWellFormedConfig(t *testing.T) {
	out, err := validateBranding(json.RawMessage(`{
		"primary_color": "#0d3d3a",
		"secondary_color": "#FFFFFF",
		"logo_url": "https://cdn.example.edu/logo.png",
		"institute_display_name": "Example Institute"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	var got brandingConfig
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.PrimaryColor == nil || *got.PrimaryColor != "#0d3d3a" {
		t.Fatalf("primary_color not preserved: %s", out)
	}
}

// The main reason this validation exists: a malicious institute admin must not
// be able to store a URL that becomes script execution in a student's browser.
func TestValidateBrandingRejectsScriptURLs(t *testing.T) {
	hostile := []string{
		`{"logo_url":"javascript:alert(1)"}`,
		`{"logo_url":"JavaScript:alert(1)"}`,
		`{"logo_url":"data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg=="}`,
		`{"logo_url":"vbscript:msgbox(1)"}`,
		`{"favicon_url":"javascript:alert(document.cookie)"}`,
		`{"logo_url":"/relative/path.png"}`,
	}
	for _, body := range hostile {
		if _, err := validateBranding(json.RawMessage(body)); err == nil {
			t.Fatalf("accepted hostile branding: %s", body)
		}
	}
}

// Colours land in a style attribute, so anything that is not a plain hex value
// is a CSS injection vector.
func TestValidateBrandingRejectsNonHexColours(t *testing.T) {
	hostile := []string{
		`{"primary_color":"red; background: url(//evil.test)"}`,
		`{"primary_color":"#fff"}`,
		`{"primary_color":"expression(alert(1))"}`,
		`{"secondary_color":"#12345g"}`,
	}
	for _, body := range hostile {
		if _, err := validateBranding(json.RawMessage(body)); err == nil {
			t.Fatalf("accepted bad colour: %s", body)
		}
	}
}

// Unknown keys are refused rather than stored, so a future frontend cannot
// start rendering a field that never passed validation.
func TestValidateBrandingRejectsUnknownKeys(t *testing.T) {
	if _, err := validateBranding(json.RawMessage(`{"custom_css":"body{display:none}"}`)); err == nil {
		t.Fatal("accepted an unknown branding key")
	}
}

func TestValidateBrandingRejectsNonObjects(t *testing.T) {
	for _, body := range []string{`5`, `"string"`, `[1,2,3]`, `null`, `not json`} {
		if _, err := validateBranding(json.RawMessage(body)); err == nil {
			t.Fatalf("accepted non-object branding: %s", body)
		}
	}
}

func TestValidateBrandingRejectsOversizedPayload(t *testing.T) {
	big := `{"institute_display_name":"` + strings.Repeat("a", maxBrandingBytes) + `"}`
	if _, err := validateBranding(json.RawMessage(big)); err == nil {
		t.Fatal("accepted an oversized branding document")
	}
}

func TestValidateBrandingEmptyIsNoOp(t *testing.T) {
	out, err := validateBranding(nil)
	if err != nil || out != nil {
		t.Fatalf("empty branding should be a no-op, got %s / %v", out, err)
	}
}

func TestValidateBrandingTreatsBlankURLAsUnset(t *testing.T) {
	out, err := validateBranding(json.RawMessage(`{"logo_url":"   "}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "logo_url") {
		t.Fatalf("blank logo_url should be dropped, got %s", out)
	}
}

func TestValidateGradingScale(t *testing.T) {
	if _, err := validateGradingScale(json.RawMessage(`{"A":90,"B":80}`)); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`[1,2]`, `"scale"`, `oops`} {
		if _, err := validateGradingScale(json.RawMessage(body)); err == nil {
			t.Fatalf("accepted bad grading scale: %s", body)
		}
	}
}

// The enum values must match the Postgres types, otherwise a valid-looking
// request fails deep in the driver as an opaque 500.
func TestValidateEnumMatchesDatabaseTypes(t *testing.T) {
	for _, v := range []string{"cgpa", "percentage", "letter"} {
		if err := validateEnum("grading_system", v, validGradingSystems); err != nil {
			t.Fatalf("%q should be a valid grading system: %v", v, err)
		}
	}
	for _, v := range []string{"gpa_4", "gpa_10", "pass_fail", ""} {
		if err := validateEnum("grading_system", v, validGradingSystems); err == nil {
			t.Fatalf("%q is not a database grading_system value", v)
		}
	}
	for _, v := range []string{"semester", "annual"} {
		if err := validateEnum("academic_calendar_type", v, validCalendarTypes); err != nil {
			t.Fatalf("%q should be a valid calendar type: %v", v, err)
		}
	}
	if err := validateEnum("academic_calendar_type", "trimester", validCalendarTypes); err == nil {
		t.Fatal("trimester is not a database academic_calendar_type value")
	}
}
