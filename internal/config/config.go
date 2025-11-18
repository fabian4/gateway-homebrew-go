package config

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

type rawConfig struct {
	EntryPoint []struct {
		Name    string `yaml:"name"`
		Address string `yaml:"address"`
	} `yaml:"entrypoint"`
	Services []struct {
		Name      string   `yaml:"name"`
		Proto     string   `yaml:"proto"`
		Endpoints []string `yaml:"endpoints"`
	} `yaml:"services"`
	Routes []struct {
		Name  string `yaml:"name"`
		Match struct {
			Hosts      []string `yaml:"hosts"`
			PathPrefix string   `yaml:"path_prefix"`
		} `yaml:"match"`
		Service string `yaml:"service"`
		Options struct {
			PreserveHost bool   `yaml:"preserve_host"`
			HostRewrite  string `yaml:"host_rewrite"`
		} `yaml:"options"`
	} `yaml:"routes"`
}

type Config struct {
	Listen   string
	Services map[string]model.Service
	Routes   []model.Route
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
		var eps []*url.URL
		for j, raw := range s.Endpoints {
			u, err := url.Parse(strings.TrimSpace(raw))
			if err != nil {
				return nil, fmt.Errorf("services[%d].endpoints[%d]: parse: %v", i, j, err)
			}
			if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return nil, fmt.Errorf("services[%d].endpoints[%d]: must be http(s) URL with host", i, j)
			}
			eps = append(eps, u)
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
		var hosts []string
		for _, h := range r.Match.Hosts {
			h = strings.ToLower(strings.TrimSpace(h))
			if h != "" {
				hosts = append(hosts, h)
			}
		}
		service := strings.TrimSpace(r.Service)
		if service == "" {
			return nil, fmt.Errorf("routes[%d]: service (service name) is required", i)
		}
		if _, ok := svcs[service]; !ok {
			return nil, fmt.Errorf("routes[%d]: service=%q not found in services", i, service)
		}
		rt := model.Route{
			Name:         name,
			Hosts:        hosts, // empty => wildcard
			PathPrefix:   pfx,
			Service:      service,
			PreserveHost: r.Options.PreserveHost,
			HostRewrite:  strings.TrimSpace(r.Options.HostRewrite),
		}
		routes = append(routes, rt)
	}
	// deterministic order: host asc ("" last), then longer prefix first
	sort.SliceStable(routes, func(i, j int) bool {
		hi := firstHost(routes[i].Hosts)
		hj := firstHost(routes[j].Hosts)
		if hi == hj {
			return len(routes[i].PathPrefix) > len(routes[j].PathPrefix)
		}
		return hi < hj
	})

	return &Config{
		Listen:   listen,
		Services: svcs,
		Routes:   routes,
	}, nil
}

func firstHost(hosts []string) string {
	if len(hosts) == 0 {
		return "~" // after normal hosts in ASCII ordering
	}
	return hosts[0]
}
