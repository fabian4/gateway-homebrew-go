package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Raw struct {
	Listen   string `yaml:"listen"`
	Upstream string `yaml:"upstream"`
}

type Parsed struct {
	Listen   string
	Upstream *url.URL
}

// Load reads YAML and returns a validated/parsed config.
func Load(path string) (*Parsed, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var r Raw
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	r.Listen = strings.TrimSpace(r.Listen)
	r.Upstream = strings.TrimSpace(r.Upstream)

	if r.Listen == "" {
		r.Listen = ":8080"
	}
	if r.Upstream == "" {
		return nil, fmt.Errorf("upstream is required")
	}

	u, err := url.Parse(r.Upstream)
	if err != nil {
		return nil, fmt.Errorf("parse upstream: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported upstream scheme: %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("upstream host is empty")
	}

	return &Parsed{
		Listen:   r.Listen,
		Upstream: u,
	}, nil
}
