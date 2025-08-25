package config

import (
	"encoding/json"
	"errors"
	"os"
)

type Config struct {
	SlackWebhook string  `json:"slack_webhook"`
	Mode         string  `json:"mode"`
	Sources      Sources `json:"sources"`
	Rules        Rules   `json:"rules"`
	RateLimit    Rate    `json:"rate_limit"`
	Batch        Batch   `json:"batch"`
	GeoIP        GeoIP   `json:"geoip"`
	Formatting   Format  `json:"formatting"`
	Telemetry    Tele    `json:"telemetry"`
}

type Sources struct {
	Prefer       string   `json:"prefer"`
	FilePaths    []string `json:"file_paths"`
	SystemdUnits []string `json:"systemd_units"`
}

type Rules struct {
	NotifySuccess     bool     `json:"notify_success"`
	NotifyFailure     bool     `json:"notify_failure"`
	NotifyInvalidUser bool     `json:"notify_invalid_user"`
	NotifyRootLogin   bool     `json:"notify_root_login"`
	ExcludeUsers      []string `json:"exclude_users"`
	ExcludeIPs        []string `json:"exclude_ips"`
	IncludeIPs        []string `json:"include_ips"`
}

type Rate struct {
	WindowSeconds      int `json:"window_seconds"`
	MaxEventsPerWindow int `json:"max_events_per_window"`
	DedupWindowSeconds int `json:"dedup_window_seconds"`
}

type Batch struct {
	WindowSeconds      int `json:"window_seconds"`
	MinFailedThreshold int `json:"min_failed_threshold"`
}

type GeoIP struct {
	Enabled bool   `json:"enabled"`
	DBPath  string `json:"db_path"`
}

type Format struct {
	Concise            bool `json:"concise"`
	ShowKeyFingerprint bool `json:"show_key_fingerprint"`
	ShowHostname       bool `json:"show_hostname"`
}

type Tele struct {
	LogLevel string `json:"log_level"`
	LogFile  string `json:"log_file"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	c.setDefaults()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) setDefaults() {
	if c.Mode == "" {
		c.Mode = "realtime"
	}
	if c.Sources.Prefer == "" {
		c.Sources.Prefer = "auto"
	}
	if len(c.Sources.SystemdUnits) == 0 {
		c.Sources.SystemdUnits = []string{"sshd.service", "ssh.service"}
	}
	if c.RateLimit.WindowSeconds == 0 {
		c.RateLimit.WindowSeconds = 60
	}
	if c.RateLimit.MaxEventsPerWindow == 0 {
		c.RateLimit.MaxEventsPerWindow = 20
	}
	if c.RateLimit.DedupWindowSeconds == 0 {
		c.RateLimit.DedupWindowSeconds = 30
	}
	if c.Telemetry.LogLevel == "" {
		c.Telemetry.LogLevel = "INFO"
	}
}

func (c *Config) validate() error {
	if c.Mode != "realtime" && c.Mode != "batch" && c.Mode != "both" && c.Mode != "" {
		return errors.New("invalid mode")
	}
	return nil
}
