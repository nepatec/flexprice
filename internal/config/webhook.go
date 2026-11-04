package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Webhook represents the configuration for the webhook system
type Webhook struct {
	Enabled               bool                           `mapstructure:"enabled"`
	Topic                 string                         `mapstructure:"topic" default:"system_events"`
	ConsumerGroup         string                         `mapstructure:"consumer_group" default:"system-events-consumer"`
	RateLimit             int64                          `mapstructure:"rate_limit" default:"5"`
	MaxRetries            int                            `mapstructure:"max_retries" default:"3"`
	InitialInterval       time.Duration                  `mapstructure:"initial_interval" default:"1s"`
	MaxInterval           time.Duration                  `mapstructure:"max_interval" default:"10s"`
	Multiplier            float64                        `mapstructure:"multiplier" default:"2.0"`
	MaxElapsedTime        time.Duration                  `mapstructure:"max_elapsed_time" default:"2m"`
	Tenants               map[string]TenantWebhookConfig `mapstructure:"tenants"`
	Svix                  Svix                           `mapstructure:"svix_config"`
	EventCascadingEnabled bool                           `mapstructure:"event_cascading_enabled" default:"false"`
}

// TenantWebhookConfig represents webhook configuration for a specific tenant.
// The json tags serve FLEXPRICE_WEBHOOK_TENANTS_JSON and must match the
// mapstructure (config.yaml) names.
type TenantWebhookConfig struct {
	Endpoint       string            `mapstructure:"endpoint" json:"endpoint"`
	Headers        map[string]string `mapstructure:"headers" json:"headers"`
	Enabled        bool              `mapstructure:"enabled" json:"enabled"`
	ExcludedEvents []string          `mapstructure:"excluded_events" json:"excluded_events"`
}

func parseTenantWebhookConfig(v *viper.Viper, envJSON string) (map[string]TenantWebhookConfig, error) {
	tenants := make(map[string]TenantWebhookConfig)
	if strings.TrimSpace(envJSON) == "" {
		if err := v.UnmarshalKey("webhook.tenants", &tenants); err != nil {
			return nil, fmt.Errorf("failed to unmarshal webhook tenants config: %v", err)
		}
		return tenants, nil
	}

	dec := json.NewDecoder(strings.NewReader(envJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&tenants); err != nil {
		return nil, fmt.Errorf("failed to parse FLEXPRICE_WEBHOOK_TENANTS_JSON: %v", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("failed to parse FLEXPRICE_WEBHOOK_TENANTS_JSON: unexpected data after the JSON object")
	}
	return tenants, nil
}

type Svix struct {
	Enabled   bool   `mapstructure:"enabled"`
	AuthToken string `mapstructure:"auth_token"`
	BaseURL   string `mapstructure:"base_url"`
}

// TenantConfig returns the webhook config for a tenant. Lookup is
// case-insensitive because Viper lowercases YAML map keys while tenant
// IDs (ULIDs) are uppercase. Always use this helper instead of indexing
// w.Tenants directly.
func (w *Webhook) TenantConfig(tenantID string) (TenantWebhookConfig, bool) {
	if w == nil || w.Tenants == nil {
		return TenantWebhookConfig{}, false
	}
	cfg, ok := w.Tenants[strings.ToLower(tenantID)]
	return cfg, ok
}

// normalizeTenantKeys rewrites Tenants so every key is lowercase. Safe to
// call multiple times. Guards against future config sources (env, remote)
// that may not lowercase keys like Viper's YAML loader does.
func (w *Webhook) normalizeTenantKeys() {
	if w == nil || len(w.Tenants) == 0 {
		return
	}
	normalized := make(map[string]TenantWebhookConfig, len(w.Tenants))
	for k, v := range w.Tenants {
		normalized[strings.ToLower(k)] = v
	}
	w.Tenants = normalized
}
