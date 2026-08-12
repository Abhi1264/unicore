package config

import (
	"strings"
	"testing"
)

// production returns a config that passes every check, so each test can break
// exactly one field and be sure that field is what caused the failure.
func production() *Config {
	return &Config{
		AppEnv:               "production",
		BaseDomain:           "unicore.app",
		WebURL:               "https://app.unicore.app",
		DatabaseURL:          "postgres://unicore_app:pw@db:5432/unicore?sslmode=require",
		JWTAccessSecret:      "a-real-access-secret-of-sufficient-length",
		JWTRefreshSecret:     "a-real-refresh-secret-of-sufficient-length",
		PaymentWebhookSecret: "a-real-webhook-secret",
		MetricsToken:         "a-real-metrics-token",
	}
}

func mustFail(t *testing.T, c *Config, want string) {
	t.Helper()
	err := c.validateSecrets()
	if err == nil {
		t.Fatalf("expected validation to fail mentioning %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not mention %q", err, want)
	}
}

func TestProductionConfigAccepted(t *testing.T) {
	if err := production().validateSecrets(); err != nil {
		t.Fatalf("a fully configured production setup should validate: %v", err)
	}
}

// The whole point of shipping working dev defaults is that they must never
// survive into production.
func TestProductionRejectsDevelopmentDefaults(t *testing.T) {
	c := production()
	c.JWTAccessSecret = DevJWTAccessSecret
	mustFail(t, c, "JWT_ACCESS_SECRET")

	c = production()
	c.JWTRefreshSecret = DevJWTRefreshSecret
	mustFail(t, c, "JWT_REFRESH_SECRET")

	c = production()
	c.PaymentWebhookSecret = DevWebhookSecret
	mustFail(t, c, "PAYMENT_WEBHOOK_SECRET")

	c = production()
	c.MetricsToken = DevMetricsToken
	mustFail(t, c, "METRICS_TOKEN")
}

func TestDevelopmentDefaultsAreUsable(t *testing.T) {
	c := &Config{
		AppEnv:           "development",
		JWTAccessSecret:  DevJWTAccessSecret,
		JWTRefreshSecret: DevJWTRefreshSecret,
	}
	if err := c.validateSecrets(); err != nil {
		t.Fatalf("development should boot with no configuration: %v", err)
	}
}

// Every dev default must clear the length bar, otherwise `docker compose up`
// fails for a first-time contributor.
func TestDevelopmentDefaultsMeetLengthRequirement(t *testing.T) {
	for name, secret := range map[string]string{
		"DevJWTAccessSecret":  DevJWTAccessSecret,
		"DevJWTRefreshSecret": DevJWTRefreshSecret,
	} {
		if len(secret) < minSecretLength {
			t.Fatalf("%s is %d chars, below the %d minimum", name, len(secret), minSecretLength)
		}
		if !isInsecureDefault(secret) {
			t.Fatalf("%s must be listed in insecureDefaults so production rejects it", name)
		}
	}
}

func TestShortSecretsRejectedEverywhere(t *testing.T) {
	c := production()
	c.AppEnv = "development"
	c.JWTAccessSecret = "tooshort"
	mustFail(t, c, "at least")
}

// Sharing one key between the two token types would let a refresh token be
// replayed as an access token.
func TestIdenticalAccessAndRefreshSecretsRejected(t *testing.T) {
	c := production()
	c.JWTRefreshSecret = c.JWTAccessSecret
	mustFail(t, c, "must differ")
}

func TestProductionRejectsRLSBypass(t *testing.T) {
	c := production()
	c.AllowRLSBypass = true
	mustFail(t, c, "ALLOW_RLS_BYPASS")
}

func TestProductionRejectsUnencryptedTransport(t *testing.T) {
	c := production()
	c.DatabaseURL = "postgres://u:p@db:5432/unicore?sslmode=disable"
	mustFail(t, c, "sslmode=disable")

	c = production()
	c.WebURL = "http://app.unicore.app"
	mustFail(t, c, "https")
}

func TestIsProduction(t *testing.T) {
	for _, env := range []string{"production", "Production", "PRODUCTION"} {
		if !(&Config{AppEnv: env}).IsProduction() {
			t.Fatalf("%q should be production", env)
		}
	}
	for _, env := range []string{"development", "staging", ""} {
		if (&Config{AppEnv: env}).IsProduction() {
			t.Fatalf("%q should not be production", env)
		}
	}
}
