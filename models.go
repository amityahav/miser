package miser

import (
	"time"
)

type Config struct {
	ESHost     string `yaml:"es_host"`
	ESUsername string `yaml:"es_username"`
	ESPassword string `yaml:"es_password"`

	SyncInterval time.Duration `yaml:"sync_interval"`
	AlertsIndex  string        `yaml:"alerts_index"`
	Notifiers    []Notifier    `yaml:"notifiers"`
}

type Notifier struct {
	Type    string `yaml:"type"`
	Name    string `yaml:"name"`
	Retries int    `yaml:"retries"`

	// Webhook configs
	Endpoint string            `yaml:"endpoint"`
	Headers  map[string]string `yaml:"headers"`
}
