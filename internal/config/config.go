package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// EndpointConfig holds connection details for one database endpoint.
type EndpointConfig struct {
	Name   string `yaml:"name"` // optional human-readable label, e.g. "DEV", "QA"
	DSN    string `yaml:"dsn"`
	Driver string `yaml:"driver"` // optional override; inferred from DSN scheme if empty
}

// IgnoreConfig specifies what to skip during comparison.
type IgnoreConfig struct {
	Tables []string `yaml:"tables"` // table names to skip entirely
	Fields []string `yaml:"fields"` // column names to skip in every table
}

// MigrateConfig controls DDL migration SQL generation.
type MigrateConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Direction  string `yaml:"direction"`   // "apply_to_target" | "apply_to_source"
	OutputFile string `yaml:"output_file"` // default: "migrate.sql"
}

// Config is the top-level configuration structure.
type Config struct {
	Source  EndpointConfig `yaml:"source"`
	Target  EndpointConfig `yaml:"target"`
	Output  string         `yaml:"output"` // "table" | "json" (default: "table")
	Schema  string         `yaml:"schema"` // optional, overrides DB name from DSN path
	Ignore  IgnoreConfig   `yaml:"ignore"`
	Migrate MigrateConfig  `yaml:"migrate"`
}

// CLIFlags holds values passed via command-line flags.
type CLIFlags struct {
	ConfigPath       string
	Source           string
	SourceName       string
	Target           string
	TargetName       string
	Output           string
	Schema           string
	IgnoreTables     string // comma-separated
	IgnoreFields     string // comma-separated
	Migrate          bool
	MigrateDirection string
	MigrateOutput    string
}

// Default returns a Config with sensible defaults applied.
func Default() *Config {
	return &Config{
		Output: "table",
		Migrate: MigrateConfig{
			Direction:  "apply_to_target",
			OutputFile: "migrate.sql",
		},
	}
}

// Load reads a YAML config file. If path is empty it auto-discovers
// "db-diff.yaml" in the current working directory. A missing auto-discovered
// file is silently ignored; an explicitly specified file that doesn't exist
// returns an error.
func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		path = "db-diff.yaml"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return cfg, nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Apply defaults for migrate output file if not set
	if cfg.Migrate.OutputFile == "" {
		cfg.Migrate.OutputFile = "migrate.sql"
	}
	if cfg.Migrate.Direction == "" {
		cfg.Migrate.Direction = "apply_to_target"
	}
	if cfg.Output == "" {
		cfg.Output = "table"
	}

	return cfg, nil
}

// Merge applies CLI flags over an existing Config. CLI flag values take
// precedence over YAML-loaded values when the flag is non-zero.
func Merge(cfg *Config, flags CLIFlags) {
	if flags.Source != "" {
		cfg.Source.DSN = flags.Source
	}
	if flags.SourceName != "" {
		cfg.Source.Name = flags.SourceName
	}
	if flags.Target != "" {
		cfg.Target.DSN = flags.Target
	}
	if flags.TargetName != "" {
		cfg.Target.Name = flags.TargetName
	}
	if flags.Output != "" {
		cfg.Output = flags.Output
	}
	if flags.Schema != "" {
		cfg.Schema = flags.Schema
	}
	if flags.IgnoreTables != "" {
		cfg.Ignore.Tables = splitComma(flags.IgnoreTables)
	}
	if flags.IgnoreFields != "" {
		cfg.Ignore.Fields = splitComma(flags.IgnoreFields)
	}
	if flags.Migrate {
		cfg.Migrate.Enabled = true
	}
	if flags.MigrateDirection != "" {
		cfg.Migrate.Direction = flags.MigrateDirection
	}
	if flags.MigrateOutput != "" {
		cfg.Migrate.OutputFile = flags.MigrateOutput
	}
}

// ResolveDriver infers the DBMS driver from the DSN scheme if not explicitly set.
// Supported prefixes: "mysql://" → "mysql", "postgres://" or "postgresql://" → "postgres".
func ResolveDriver(ep *EndpointConfig) error {
	if ep.Driver != "" {
		return nil
	}
	dsn := ep.DSN
	switch {
	case strings.HasPrefix(dsn, "mysql://"):
		ep.Driver = "mysql"
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		ep.Driver = "postgres"
	default:
		return fmt.Errorf("cannot infer driver from DSN %q: use --source-driver / --target-driver or add 'driver:' to config", dsn)
	}
	return nil
}

// splitComma splits a comma-separated string and trims whitespace.
func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
