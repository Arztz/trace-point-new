package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the entire application configuration.
type Config struct {
	App         AppConfig         `yaml:"app"`
	Datasources []DatasourceConfig `yaml:"datasources"`
	Detection   DetectionConfig   `yaml:"detection"`
	Timeline    TimelineConfig    `yaml:"timeline"`
	Discord     DiscordConfig     `yaml:"discord"`
	Database    DatabaseConfig    `yaml:"database"`
}

type DatasourceConfig struct {
	Name                    string           `yaml:"name"`
	Prometheus              PrometheusConfig `yaml:"prometheus"`
	Signoz                  SignozConfig     `yaml:"signoz"`
	GCloud                  GCloudConfig     `yaml:"gcloud"`
	Namespaces              []string         `yaml:"namespaces"`
	DeployExcludePatterns   []string         `yaml:"deploy_exclude_patterns"`
	
	// Compiled regex patterns (not from YAML)
	CompiledExcludePatterns []*regexp.Regexp `yaml:"-"`
}

type AppConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type PrometheusConfig struct {
	URL                      string        `yaml:"url"`
	Timeout                  time.Duration `yaml:"timeout"`
	UseDeploymentAggregation bool          `yaml:"use_deployment_aggregation"`
}

type SignozConfig struct {
	URL      string        `yaml:"url"`
	User     string        `yaml:"user"`
	Password string        `yaml:"password"`
	Timeout  time.Duration `yaml:"timeout"`
	Database string        `yaml:"database"`
	EnvTag   string        `yaml:"env_tag"`
}

type GCloudConfig struct {
	ProjectID       string `yaml:"project_id"`
	EnvVersionTag   string `yaml:"env_version_tag"`
	ProfilerEnabled bool   `yaml:"profiler_enabled"`
}

type DetectionConfig struct {
	CPUThreshold                float64 `yaml:"cpu_threshold"`
	MemoryThreshold             float64 `yaml:"memory_threshold"`
	PollingIntervalSeconds      int     `yaml:"polling_interval_seconds"`
	MovingAverageWindowMinutes  int     `yaml:"moving_average_window_minutes"`
	BaselineLearningPeriodMinutes int   `yaml:"baseline_learning_period_minutes"`
	ReconciliationBufferMinutes int     `yaml:"reconciliation_buffer_minutes"`
	CooldownMinutes             int     `yaml:"cooldown_minutes"`
}

type TimelineConfig struct {
	CPUCloseTo100Threshold  float64 `yaml:"cpu_close_to_100_threshold"`
	CPUFarBelow100Threshold float64 `yaml:"cpu_far_below_100_threshold"`
	RAMCloseTo100Threshold  float64 `yaml:"ram_close_to_100_threshold"`
	RAMFarBelow100Threshold float64 `yaml:"ram_far_below_100_threshold"`
}

type DiscordConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

type DatabaseConfig struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

// Load reads the configuration from the specified YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the YAML content
	expanded := expandEnvVars(string(data))

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.setDefaults()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if err := cfg.compilePatterns(); err != nil {
		return nil, fmt.Errorf("failed to compile exclude patterns: %w", err)
	}

	return cfg, nil
}

// expandEnvVars replaces ${VAR} patterns with environment variable values.
func expandEnvVars(s string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match
	})
}

func (c *Config) setDefaults() {
	if c.App.Host == "" {
		c.App.Host = "0.0.0.0"
	}
	if c.App.Port == 0 {
		c.App.Port = 8088
	}
	
	for i := range c.Datasources {
		ds := &c.Datasources[i]
		if ds.Prometheus.Timeout == 0 {
			ds.Prometheus.Timeout = 30 * time.Second
		}
		// Always use deployment aggregation
		ds.Prometheus.UseDeploymentAggregation = true

		if ds.Signoz.Timeout == 0 {
			ds.Signoz.Timeout = 30 * time.Second
		}
		if ds.Signoz.Database == "" {
			ds.Signoz.Database = "signoz_traces"
		}
	}
	if c.Detection.CPUThreshold == 0 {
		c.Detection.CPUThreshold = 80
	}
	if c.Detection.MemoryThreshold == 0 {
		c.Detection.MemoryThreshold = 85
	}
	if c.Detection.PollingIntervalSeconds == 0 {
		c.Detection.PollingIntervalSeconds = 30
	}
	if c.Detection.MovingAverageWindowMinutes == 0 {
		c.Detection.MovingAverageWindowMinutes = 30
	}
	if c.Detection.BaselineLearningPeriodMinutes == 0 {
		c.Detection.BaselineLearningPeriodMinutes = 5
	}
	if c.Detection.ReconciliationBufferMinutes == 0 {
		c.Detection.ReconciliationBufferMinutes = 8
	}
	if c.Detection.CooldownMinutes == 0 {
		c.Detection.CooldownMinutes = 15
	}
	if c.Timeline.CPUCloseTo100Threshold == 0 {
		c.Timeline.CPUCloseTo100Threshold = 85
	}
	if c.Timeline.CPUFarBelow100Threshold == 0 {
		c.Timeline.CPUFarBelow100Threshold = 50
	}
	if c.Timeline.RAMCloseTo100Threshold == 0 {
		c.Timeline.RAMCloseTo100Threshold = 85
	}
	if c.Timeline.RAMFarBelow100Threshold == 0 {
		c.Timeline.RAMFarBelow100Threshold = 50
	}
	if c.Database.Type == "" {
		c.Database.Type = "sqlite"
	}
	if c.Database.Path == "" {
		c.Database.Path = "./data/trace-point.db"
	}
}

func (c *Config) validate() error {
	if len(c.Datasources) == 0 {
		return fmt.Errorf("at least one datasource must be configured")
	}
	for i, ds := range c.Datasources {
		if ds.Name == "" {
			return fmt.Errorf("datasource %d must have a name", i)
		}
		if ds.Prometheus.URL == "" {
			return fmt.Errorf("datasource %s prometheus.url is required", ds.Name)
		}
		if len(ds.Namespaces) == 0 {
			return fmt.Errorf("datasource %s requires at least one namespace", ds.Name)
		}
	}
	return nil
}

func (c *Config) compilePatterns() error {
	for i := range c.Datasources {
		ds := &c.Datasources[i]
		ds.CompiledExcludePatterns = make([]*regexp.Regexp, 0, len(ds.DeployExcludePatterns))
		for _, pattern := range ds.DeployExcludePatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid exclude pattern %q in datasource %s: %w", pattern, ds.Name, err)
			}
			ds.CompiledExcludePatterns = append(ds.CompiledExcludePatterns, re)
		}
	}
	return nil
}

// ShouldExcludeDeployment checks if a deployment name matches any exclude pattern for a specific datasource.
func (ds *DatasourceConfig) ShouldExcludeDeployment(name string) bool {
	for _, re := range ds.CompiledExcludePatterns {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// GetListenAddr returns the formatted listen address.
func (c *Config) GetListenAddr() string {
	return fmt.Sprintf("%s:%d", c.App.Host, c.App.Port)
}

// GetNamespacesRegex returns namespaces joined as a regex alternation for a specific datasource.
func (ds *DatasourceConfig) GetNamespacesRegex() string {
	return strings.Join(ds.Namespaces, "|")
}
