package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type StoreConfig struct {
	Type       string `yaml:"type"`
	TableName  string `default:"certs" yaml:"tableName,omitempty"`
	ConnString string `yaml:"uri,omitempty"`
	Flush      bool   `default:"true" yaml:"flush,omitempty"`
}

type NotificationsConfig struct {
	Type       string   `yaml:"type"`
	Recipients []string `yaml:"recipients"`
}

type SimilarityCheckCfg struct {
	Distance float64 `default:"0.75" yaml:"similarity_distance,omitempty"`
}

// DaemonConfig holds the optional "daemon:" section. Fields are pointers so an
// absent key is distinguishable from a zero value: when set, they take
// precedence over the equivalent command-line flags (see cmd.RunAsDaemon).
type DaemonConfig struct {
	RescanInterval *int  `yaml:"rescanInterval,omitempty"`
	Foreground     *bool `yaml:"foreground,omitempty"`
}

type CheckConfig struct {
	Name           string             `yaml:"name"`
	Regex          []string           `yaml:"regex,omitempty"`
	Patterns       []string           `yaml:"patterns,omitempty"`
	Logs           []string           `yaml:"logs,omitempty"`
	LookupDepth    int64              `yaml:"lookupDepth"` // lookback window in seconds
	LookupDelta    int64              `yaml:"lookupDelta,omitempty"`
	RescanInterval int                `default:"60" yaml:"rescanInterval"`
	WorkersCount   int                `default:"1" yaml:"numWorkers"`
	IncludeEntry   bool               `yaml:"includeEntry,omitempty"`
	SimilarityCfg  SimilarityCheckCfg `yaml:"similarity,omitempty"`
}

type Config struct {
	Version        int                   `yaml:"version"`
	Verbose        bool                  `default:"true" yaml:"verbose"`
	State          bool                  `default:"true" yaml:"state"`
	WorkersCount   int                   `default:"1" yaml:"numWorkers"`
	BatchSize      int                   `default:"100" yaml:"batchSize"`
	CtLogs         []string              `yaml:"defaultCtLogs"`
	Checks         []CheckConfig         `yaml:"checks"`
	Notifications  []NotificationsConfig `yaml:"notifications"`
	Store          []StoreConfig         `yaml:"store,omitempty"`
	RescanInterval int                   `default:"60" yaml:"rescanInterval,omitempty"`
	Daemon         DaemonConfig          `yaml:"daemon,omitempty"`
}

func NewConfig(content []byte) (*Config, error) {
	var cfg Config
	marshallErr := yaml.Unmarshal(content, &cfg)
	if marshallErr != nil {
		panic(marshallErr)
	}
	return &cfg, nil
}

func NewConfigFile(cfgPath string) (*Config, error) {
	yamlFile, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		panic(readErr)
	}
	return NewConfig(yamlFile)
}

func (c *Config) GetNotificationsCfg() []NotificationsConfig {
	return c.Notifications
}

func (c *Config) GetStoreConfigs() []StoreConfig {
	return c.Store
}

func (c *Config) IsVerbose() bool {
	return c.Verbose
}

func (c *Config) GetChecksCfg() []CheckConfig {
	return c.Checks
}

// IsDaemon reports whether the config requests daemon mode, inferred from the
// presence of any field in the daemon section.
func (c *Config) IsDaemon() bool {
	return c.Daemon.RescanInterval != nil || c.Daemon.Foreground != nil
}

func (c *Config) EnableHealthCheck() bool {
	// assuming that we enable healthcheck for daemon by default
	return c.IsDaemon()
}
