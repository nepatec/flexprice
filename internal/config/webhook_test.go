package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestWebhook_TenantConfig_NilReceiver(t *testing.T) {
	t.Parallel()

	var w *Webhook
	cfg, ok := w.TenantConfig("tenant_01KE93H5A5S0S3DMBJT4YP589M")
	require.False(t, ok)
	require.Equal(t, TenantWebhookConfig{}, cfg)
}

func TestWebhook_TenantConfig_NilMap(t *testing.T) {
	t.Parallel()

	w := &Webhook{}
	cfg, ok := w.TenantConfig("tenant_01KE93H5A5S0S3DMBJT4YP589M")
	require.False(t, ok)
	require.Equal(t, TenantWebhookConfig{}, cfg)
}

func TestWebhook_TenantConfig_CaseInsensitiveLookup(t *testing.T) {
	t.Parallel()

	// Mimic Viper's post-load state: map key is lowercase. Real tenant IDs
	// arrive uppercase (ULID / Crockford Base32), so the lookup must
	// normalize before indexing.
	w := &Webhook{
		Tenants: map[string]TenantWebhookConfig{
			"tenant_01ke93h5a5s0s3dmbjt4yp589m": {
				Enabled:  true,
				Endpoint: "http://localhost:8080/health",
			},
		},
	}

	cfg, ok := w.TenantConfig("tenant_01KE93H5A5S0S3DMBJT4YP589M")
	require.True(t, ok)
	require.True(t, cfg.Enabled)
	require.Equal(t, "http://localhost:8080/health", cfg.Endpoint)
}

func TestWebhook_TenantConfig_Miss(t *testing.T) {
	t.Parallel()

	w := &Webhook{
		Tenants: map[string]TenantWebhookConfig{
			"tenant_known": {Enabled: true},
		},
	}

	_, ok := w.TenantConfig("tenant_unknown")
	require.False(t, ok)
}

func TestWebhook_normalizeTenantKeys(t *testing.T) {
	t.Parallel()

	w := &Webhook{
		Tenants: map[string]TenantWebhookConfig{
			"tenant_01KE93H5A5S0S3DMBJT4YP589M": {Enabled: true, Endpoint: "a"},
			"already_lowercase":                 {Enabled: false, Endpoint: "b"},
		},
	}

	w.normalizeTenantKeys()

	require.Len(t, w.Tenants, 2)
	upper, ok := w.Tenants["tenant_01ke93h5a5s0s3dmbjt4yp589m"]
	require.True(t, ok)
	require.Equal(t, "a", upper.Endpoint)
	lower, ok := w.Tenants["already_lowercase"]
	require.True(t, ok)
	require.Equal(t, "b", lower.Endpoint)

	// Idempotent.
	require.NotPanics(t, func() { w.normalizeTenantKeys() })
	require.Len(t, w.Tenants, 2)
}

func TestWebhook_normalizeTenantKeys_NilAndEmptySafe(t *testing.T) {
	t.Parallel()

	var nilWebhook *Webhook
	require.NotPanics(t, func() { nilWebhook.normalizeTenantKeys() })

	emptyWebhook := &Webhook{}
	require.NotPanics(t, func() { emptyWebhook.normalizeTenantKeys() })
	require.Nil(t, emptyWebhook.Tenants)
}

func TestParseTenantWebhookConfig_FromEnvJSON(t *testing.T) {
	t.Parallel()

	v := viper.New()
	v.Set("webhook.tenants", map[string]any{
		"tenant_yaml": map[string]any{"enabled": true, "endpoint": "https://yaml.example"},
	})

	got, err := parseTenantWebhookConfig(v, `{
		"tenant_01KE93H5A5S0S3DMBJT4YP589M": {
			"enabled": true,
			"endpoint": "https://env.example/webhooks",
			"headers": {"X-Api-Key": "secret"},
			"excluded_events": ["tenant.created", "tenant.updated"]
		}
	}`)
	require.NoError(t, err)
	// The env value replaces the YAML map entirely (no merge).
	require.Equal(t, map[string]TenantWebhookConfig{
		"tenant_01KE93H5A5S0S3DMBJT4YP589M": {
			Enabled:        true,
			Endpoint:       "https://env.example/webhooks",
			Headers:        map[string]string{"X-Api-Key": "secret"},
			ExcludedEvents: []string{"tenant.created", "tenant.updated"},
		},
	}, got)
}

func TestParseTenantWebhookConfig_EmptyEnvFallsBackToYAML(t *testing.T) {
	t.Parallel()

	v := viper.New()
	v.Set("webhook.tenants", map[string]any{
		"tenant_yaml": map[string]any{"enabled": true, "endpoint": "https://yaml.example"},
	})

	for _, envJSON := range []string{"", "   "} {
		got, err := parseTenantWebhookConfig(v, envJSON)
		require.NoError(t, err)
		require.Equal(t, map[string]TenantWebhookConfig{
			"tenant_yaml": {Enabled: true, Endpoint: "https://yaml.example"},
		}, got)
	}
}

func TestParseTenantWebhookConfig_EmptyObjectDisablesAllTenants(t *testing.T) {
	t.Parallel()

	v := viper.New()
	v.Set("webhook.tenants", map[string]any{
		"tenant_yaml": map[string]any{"enabled": true, "endpoint": "https://yaml.example"},
	})

	got, err := parseTenantWebhookConfig(v, `{}`)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestParseTenantWebhookConfig_RejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	for name, envJSON := range map[string]string{
		"malformed":     `{"tenant_a": {`,
		"unknown field": `{"tenant_a": {"enabled": true, "excludedEvents": ["tenant.created"]}}`,
		"wrong type":    `{"tenant_a": {"enabled": "yes"}}`,
		"trailing data": `{"tenant_a": {"enabled": true}} {}`,
		"not an object": `["tenant_a"]`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseTenantWebhookConfig(viper.New(), envJSON)
			require.ErrorContains(t, err, "FLEXPRICE_WEBHOOK_TENANTS_JSON")
		})
	}
}

// TestNewConfig_WebhookTenantsFromEnvJSON covers the wiring in NewConfig,
// including tenant-key normalization so TenantConfig lookups still match.
func TestNewConfig_WebhookTenantsFromEnvJSON(t *testing.T) {
	t.Setenv("FLEXPRICE_WEBHOOK_TENANTS_JSON",
		`{"tenant_01KE93H5A5S0S3DMBJT4YP589M":{"enabled":true,"endpoint":"https://env.example/webhooks","headers":{"X-Api-Key":"secret"}}}`)

	cfg, err := NewConfig()
	require.NoError(t, err)
	require.Len(t, cfg.Webhook.Tenants, 1)

	tc, ok := cfg.Webhook.TenantConfig("tenant_01KE93H5A5S0S3DMBJT4YP589M")
	require.True(t, ok)
	require.True(t, tc.Enabled)
	require.Equal(t, "https://env.example/webhooks", tc.Endpoint)
	require.Equal(t, "secret", tc.Headers["X-Api-Key"])
}

func TestNewConfig_InvalidWebhookTenantsEnvJSONFails(t *testing.T) {
	t.Setenv("FLEXPRICE_WEBHOOK_TENANTS_JSON", `{"tenant_a":`)

	_, err := NewConfig()
	require.ErrorContains(t, err, "FLEXPRICE_WEBHOOK_TENANTS_JSON")
}
