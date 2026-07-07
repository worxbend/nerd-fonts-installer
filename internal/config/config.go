package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/worxbend/nerd-fonts-installer/internal/fontname"
	"gopkg.in/yaml.v3"
)

const appName = "nerd-fonts-installer"

var configExtensions = []string{".yaml", ".yml", ".json", ".conf"}

type Config struct {
	Release          string   `json:"release" yaml:"release"`
	Destination      string   `json:"destination" yaml:"destination"`
	RefreshFontCache bool     `json:"refresh_font_cache" yaml:"refresh_font_cache"`
	Families         []string `json:"families" yaml:"families"`
}

type Source struct {
	Path   string
	Config Config
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := decode(path, data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.ApplyDefaults()
	cfg.Normalize()
	return cfg, cfg.Validate()
}

func decode(path string, data []byte, cfg *Config) error {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(cfg); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if err := decoder.Decode(&struct{}{}); err == nil {
			return fmt.Errorf("parse %s: multiple json values", path)
		} else if !errors.Is(err, io.EOF) {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		return nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func (c *Config) ApplyDefaults() {
	if c.Release == "" {
		c.Release = "latest"
	}
	if c.Destination == "" {
		c.Destination = "~/.local/share/fonts/NerdFonts"
	}
}

func (c *Config) Normalize() {
	c.Release = strings.TrimSpace(c.Release)
	c.Destination = strings.TrimSpace(c.Destination)
	for i, family := range c.Families {
		c.Families[i] = strings.TrimSpace(family)
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Release) == "" {
		return fmt.Errorf("release is required")
	}
	if strings.TrimSpace(c.Destination) == "" {
		return fmt.Errorf("destination is required")
	}
	if len(c.Families) == 0 {
		return fmt.Errorf("at least one font family is required")
	}
	seen := map[string]bool{}
	for _, family := range c.Families {
		family = strings.TrimSpace(family)
		if err := fontname.Validate(family); err != nil {
			return err
		}
		if seen[family] {
			return fmt.Errorf("duplicate font family %q", family)
		}
		seen[family] = true
	}
	return nil
}

func Discover() (Source, bool, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return Source{}, false, err
	}
	return DiscoverPaths(paths)
}

func DiscoverPaths(paths []string) (Source, bool, error) {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		cfg, err := Load(path)
		if err == nil {
			return Source{Path: path, Config: cfg}, true, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return Source{}, false, fmt.Errorf("load discovered config %s: %w", path, err)
	}
	return Source{}, false, nil
}

func DefaultPaths() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("locate current directory: %w", err)
	}

	paths := []string{}
	paths = appendConfigCandidates(paths, cwd)
	if configHome, ok := userConfigHome(); ok {
		paths = appendConfigCandidates(paths, configHome)
	}
	return paths, nil
}

func appendConfigCandidates(paths []string, baseDir string) []string {
	for _, extension := range configExtensions {
		paths = appendUnique(paths, filepath.Join(baseDir, appName+extension))
	}
	for _, extension := range configExtensions {
		paths = appendUnique(paths, filepath.Join(baseDir, appName, "config"+extension))
	}
	return paths
}

func appendUnique(paths []string, path string) []string {
	for _, existing := range paths {
		if existing == path {
			return paths
		}
	}
	return append(paths, path)
}

func userConfigHome() (string, bool) {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(xdgConfigHome) {
		return xdgConfigHome, true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".config"), true
}
