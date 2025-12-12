package config

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

type rawConfig struct {
	EntryPoint []struct {
		Name    string `yaml:"name"`
		Address string `yaml:"address"`
	} `yaml:"entrypoint"`
	Services []struct {
		Name      string `yaml:"name"`
		Proto     string `yaml:"proto"`
		Endpoints []any  `yaml:"endpoints"`
	} `yaml:"services"`
	Routes []struct {
		Name  string `yaml:"name"`
		Match struct {
			Host       string `yaml:"host"`
			PathPrefix string `yaml:"path_prefix"`
		} `yaml:"match"`
		Service string `yaml:"service"`
		Options struct {
			PreserveHost bool   `yaml:"preserve_host"`
			HostRewrite  string `yaml:"host_rewrite"`
		} `yaml:"options"`
	} `yaml:"routes"`
	Timeouts struct {
		Read     string `yaml:"read"`
		Write    string `yaml:"write"`
		Upstream string `yaml:"upstream"`
	} `yaml:"timeouts"`
}

type Config struct {
	Listen   string
	Services map[string]model.Service
	Routes   []model.Route
	Timeouts Timeouts
}

type Timeouts struct {
	Read     time.Duration
	Write    time.Duration
	Upstream time.Duration
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var rc rawConfig
	if err := yaml.Unmarshal(b, &rc); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}

	// listen
	listen := ":8080"
	if len(rc.EntryPoint) > 0 && strings.TrimSpace(rc.EntryPoint[0].Address) != "" {
		listen = strings.TrimSpace(rc.EntryPoint[0].Address)
	}

	// services
	svcs := make(map[string]model.Service)
	for i, s := range rc.Services {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			return nil, fmt.Errorf("services[%d]: name is required", i)
		}
		proto := strings.ToLower(strings.TrimSpace(s.Proto))
		if proto == "" {
			proto = "http1"
		}
		switch proto {
		case "http1", "auto", "h2c":
		default:
			return nil, fmt.Errorf("services[%d]: unknown proto %q", i, proto)
		}
		if len(s.Endpoints) == 0 {
			return nil, fmt.Errorf("services[%d]: endpoints is empty", i)
		}
		var eps []model.Endpoint
		for j, raw := range s.Endpoints {
			var rawURL string
			weight := 1

			switch v := raw.(type) {
			case string:
				rawURL = v
			case map[string]any:
				if u, ok := v["url"].(string); ok {
					rawURL = u
				}
				if w, ok := v["weight"].(int); ok {
					weight = w
				}
			default:
				return nil, fmt.Errorf("services[%d].endpoints[%d]: invalid format", i, j)
			}

			u, err := url.Parse(strings.TrimSpace(rawURL))
			if err != nil {
				return nil, fmt.Errorf("services[%d].endpoints[%d]: parse: %v", i, j, err)
			}
			if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return nil, fmt.Errorf("services[%d].endpoints[%d]: must be http(s) URL with host", i, j)
			}
			eps = append(eps, model.Endpoint{URL: u, Weight: weight})
		}
		if _, dup := svcs[name]; dup {
			return nil, fmt.Errorf("services: duplicate name %q", name)
		}
		svcs[name] = model.Service{
			Name:      name,
			Proto:     proto,
			Endpoints: eps,
		}
	}
	if len(svcs) == 0 {
		return nil, fmt.Errorf("services: at least one is required")
	}

	// routes
	var routes []model.Route
	for i, r := range rc.Routes {
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = fmt.Sprintf("route-%d", i)
		}
		pfx := strings.TrimSpace(r.Match.PathPrefix)
		if !strings.HasPrefix(pfx, "/") {
			return nil, fmt.Errorf("routes[%d]: path_prefix must start with '/'", i)
		}
		host := strings.ToLower(strings.TrimSpace(r.Match.Host))
		service := strings.TrimSpace(r.Service)
		if service == "" {
			return nil, fmt.Errorf("routes[%d]: service (service name) is required", i)
		}
		if _, ok := svcs[service]; !ok {
			return nil, fmt.Errorf("routes[%d]: service=%q not found in services", i, service)
		}
		rt := model.Route{
			Name:         name,
			Host:         host, // empty => wildcard
			PathPrefix:   pfx,
			Service:      service,
			PreserveHost: r.Options.PreserveHost,
			HostRewrite:  strings.TrimSpace(r.Options.HostRewrite),
		}
		routes = append(routes, rt)
	}
	// deterministic order: host asc ("" last), then longer prefix first
	sort.SliceStable(routes, func(i, j int) bool {
		hi := routes[i].Host
		hj := routes[j].Host
		if hi == "" {
			hi = "~"
		}
		if hj == "" {
			hj = "~"
		}
		if hi == hj {
			return len(routes[i].PathPrefix) > len(routes[j].PathPrefix)
		}
		return hi < hj
	})

	// timeouts
	var timeouts Timeouts
	if rc.Timeouts.Read != "" {
		d, err := time.ParseDuration(rc.Timeouts.Read)
		if err != nil {
			return nil, fmt.Errorf("timeouts.read: %v", err)
		}
		timeouts.Read = d
	}
	if rc.Timeouts.Write != "" {
		d, err := time.ParseDuration(rc.Timeouts.Write)
		if err != nil {
			return nil, fmt.Errorf("timeouts.write: %v", err)
		}
		timeouts.Write = d
	}
	if rc.Timeouts.Upstream != "" {
		d, err := time.ParseDuration(rc.Timeouts.Upstream)
		if err != nil {
			return nil, fmt.Errorf("timeouts.upstream: %v", err)
		}
		timeouts.Upstream = d
	}

	return &Config{
		Listen:   listen,
		Services: svcs,
		Routes:   routes,
		Timeouts: timeouts,
	}, nil
}
