package config

import "testing"

func TestEffectiveTokenFromConfig(t *testing.T) {
	t.Setenv(EnvToken, "")
	t.Setenv(LegacyEnvToken, "")

	token, source := EffectiveToken(Config{Token: "config-token"})
	if token != "config-token" {
		t.Fatalf("unexpected token: %s", token)
	}
	if source != "config" {
		t.Fatalf("unexpected source: %s", source)
	}
}

func TestEffectiveTokenFromPrimaryEnv(t *testing.T) {
	t.Setenv(EnvToken, "env-token")
	t.Setenv(LegacyEnvToken, "legacy-token")

	token, source := EffectiveToken(Config{Token: "config-token"})
	if token != "env-token" {
		t.Fatalf("unexpected token: %s", token)
	}
	if source != "env:"+EnvToken {
		t.Fatalf("unexpected source: %s", source)
	}
}

func TestEffectiveTokenFromLegacyEnv(t *testing.T) {
	t.Setenv(EnvToken, "")
	t.Setenv(LegacyEnvToken, "legacy-token")

	token, source := EffectiveToken(Config{Token: "config-token"})
	if token != "legacy-token" {
		t.Fatalf("unexpected token: %s", token)
	}
	if source != "env:"+LegacyEnvToken {
		t.Fatalf("unexpected source: %s", source)
	}
}
