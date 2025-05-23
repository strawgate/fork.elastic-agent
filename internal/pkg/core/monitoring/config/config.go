// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

package config

import (
	"strings"
	"time"

	c "github.com/elastic/elastic-agent-libs/config"
)

const (
	defaultPort      = 6791
	defaultNamespace = "default"

	// DefaultHost is used when host is not defined or empty
	DefaultHost           = "localhost"
	ProcessRuntimeManager = "process"
	OtelRuntimeManager    = "otel"
	DefaultRuntimeManager = ProcessRuntimeManager
)

// MonitoringConfig describes a configuration of a monitoring
type MonitoringConfig struct {
	Enabled          bool                  `yaml:"enabled" config:"enabled"`
	MonitorLogs      bool                  `yaml:"logs" config:"logs"`
	MonitorMetrics   bool                  `yaml:"metrics" config:"metrics"`
	MetricsPeriod    string                `yaml:"metrics_period" config:"metrics_period"`
	FailureThreshold *uint                 `yaml:"failure_threshold" config:"failure_threshold"`
	LogMetrics       bool                  `yaml:"-" config:"-"`
	HTTP             *MonitoringHTTPConfig `yaml:"http" config:"http"`
	Namespace        string                `yaml:"namespace" config:"namespace"`
	Pprof            *PprofConfig          `yaml:"pprof" config:"pprof"`
	MonitorTraces    bool                  `yaml:"traces" config:"traces"`
	APM              APMConfig             `yaml:"apm,omitempty" config:"apm,omitempty" json:"apm,omitempty"`
	Diagnostics      Diagnostics           `yaml:"diagnostics,omitempty" json:"diagnostics,omitempty"`
	RuntimeManager   string                `yaml:"_runtime_experimental,omitempty" config:"_runtime_experimental,omitempty"`
}

// MonitoringHTTPConfig is a config defining HTTP endpoint published by agent
// for other processes to watch its metrics.
// Processes are only exposed when HTTP is enabled.
type MonitoringHTTPConfig struct {
	Enabled bool          `yaml:"enabled" config:"enabled"`
	Host    string        `yaml:"host" config:"host"`
	Port    int           `yaml:"port" config:"port" validate:"min=0,max=65535"`
	Buffer  *BufferConfig `yaml:"buffer" config:"buffer"`
	// EnabledIsSet is set during the Unpack() operation, and will be set to true if `Enabled` has been manually set by the incoming yaml
	// This is done so we can distinguish between a default value supplied by the code, and a user-supplied value
	EnabledIsSet bool `yaml:"-" config:"-"`
}

// Unpack reads a config object into the settings.
func (c *MonitoringHTTPConfig) Unpack(cfg *c.C) error {
	// do not use MonitoringHTTPConfig, it will end up in a loop
	tmp := struct {
		Enabled *bool         `yaml:"enabled" config:"enabled"`
		Host    string        `yaml:"host" config:"host"`
		Port    int           `yaml:"port" config:"port" validate:"min=0,max=65535"`
		Buffer  *BufferConfig `yaml:"buffer" config:"buffer"`
	}{
		Host:   c.Host,
		Port:   c.Port,
		Buffer: c.Buffer,
	}

	if err := cfg.Unpack(&tmp); err != nil {
		return err
	}

	if strings.TrimSpace(tmp.Host) == "" {
		tmp.Host = DefaultHost
	}

	set := MonitoringHTTPConfig{
		Host:   tmp.Host,
		Port:   tmp.Port,
		Buffer: tmp.Buffer,
	}

	// this logic is here to help us distinguish between `http.enabled` being manually set after unpacking,
	// and whatever a user-specified default may be.
	// This is needed in order to prevent a larger set of breaking changes where fleet doesn't expect the HTTP monitor to be live-reloadable
	// see https://github.com/elastic/elastic-agent/issues/4582
	if tmp.Enabled == nil {
		set.EnabledIsSet = false
		set.Enabled = c.Enabled
	} else {
		set.EnabledIsSet = true
		set.Enabled = *tmp.Enabled
	}

	*c = set

	return nil
}

// PprofConfig is a struct for the pprof enablement flag.
// It is a nil struct by default to allow the agent to use the a value that the user has injected into fleet.yml as the source of truth that is passed to beats
// TODO get this value from Kibana?
type PprofConfig struct {
	Enabled bool `yaml:"enabled" config:"enabled"`
}

// BufferConfig is a struct for for the metrics buffer endpoint
type BufferConfig struct {
	Enabled bool `yaml:"enabled" config:"enabled"`
}

// DefaultConfig creates a config with pre-set default values.
func DefaultConfig() *MonitoringConfig {
	return &MonitoringConfig{
		Enabled:        true,
		MonitorLogs:    true,
		MonitorMetrics: true,
		LogMetrics:     true,
		MonitorTraces:  false,
		HTTP: &MonitoringHTTPConfig{
			Enabled: false,
			Host:    DefaultHost,
			Port:    defaultPort,
		},
		Namespace:      defaultNamespace,
		APM:            defaultAPMConfig(),
		Diagnostics:    defaultDiagnostics(),
		RuntimeManager: DefaultRuntimeManager,
	}
}

// APMConfig configures APM Tracing.
type APMConfig struct {
	Environment  string            `config:"environment" yaml:"environment,omitempty"`
	APIKey       string            `config:"api_key" yaml:"api_key,omitempty"`
	SecretToken  string            `config:"secret_token" yaml:"secret_token,omitempty"`
	Hosts        []string          `config:"hosts" yaml:"hosts,omitempty"`
	GlobalLabels map[string]string `config:"global_labels" yaml:"global_labels,omitempty"`
	TLS          APMTLS            `config:"tls" yaml:"tls,omitempty"`
	SamplingRate *float32          `config:"sampling_rate" yaml:"sampling_rate,omitempty"`
}

// APMTLS contains the configuration options necessary for configuring TLS in
// apm-agent-go.
type APMTLS struct {
	SkipVerify        bool   `config:"skip_verify" yaml:"skip_verify,omitempty"`
	ServerCertificate string `config:"server_certificate" yaml:"server_certificate,omitempty"`
	ServerCA          string `config:"server_ca" yaml:"server_ca,omitempty"`
}

func defaultAPMConfig() APMConfig {
	return APMConfig{}
}

// Uploader contains the configuration for retries when uploading a file (diagnostics bundle) to fleet-server.
type Uploader struct {
	MaxRetries int           `config:"max_retries"`
	InitDur    time.Duration `config:"init_duration"`
	MaxDur     time.Duration `config:"max_duration"`
}

func defaultUploader() Uploader {
	return Uploader{
		MaxRetries: 10,
		InitDur:    time.Second,
		MaxDur:     time.Minute * 10,
	}
}

// Limit contains the configuration for rate-limiting operations
type Limit struct {
	Interval time.Duration `config:"interval"`
	Burst    int           `config:"burst"`
}

func defaultLimit() Limit {
	return Limit{
		Interval: time.Minute,
		Burst:    1,
	}
}

// Diagnostics contains the configuration needed to configure the diagnostics handler.
type Diagnostics struct {
	Uploader Uploader `config:"uploader"`
	Limit    Limit    `config:"limit"`
}

func defaultDiagnostics() Diagnostics {
	return Diagnostics{
		Uploader: defaultUploader(),
		Limit:    defaultLimit(),
	}
}
