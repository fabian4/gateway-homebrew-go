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

type Raw struct {
	Listen string  `yaml:"listen"`
	Routes []Route `yaml:"routes"`
}

type Route struct {
	Host     string `yaml:"host"`
	Prefix   string `yaml:"prefix"`
	Upstream string `yaml:"upstream"`
	Proto    string `yaml:"proto"` // optional: http1 | auto | h2c
}

type Parsed struct {
	Listen string
	Routes []model.Route
}

func Load(path string) (*Parsed, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var r Raw
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	if strings.TrimSpace(r.Listen) == "" {
		r.Listen = ":8080"
	}
	if len(r.Routes) == 0 {
		return nil, fmt.Errorf("routes is required (at least one)")
	}

	out := &Parsed{Listen: r.Listen}
	for i, rt := range r.Routes {
		host := strings.ToLower(strings.TrimSpace(rt.Host))
		pfx := strings.TrimSpace(rt.Prefix)
		if !strings.HasPrefix(pfx, "/") {
			return nil, fmt.Errorf("routes[%d]: prefix must start with '/'", i)
		}

		us := strings.TrimSpace(rt.Upstream)
		if us == "" {
			return nil, fmt.Errorf("routes[%d]: upstream is required", i)
		}
		u, err := url.Parse(us)
		if err != nil {
			return nil, fmt.Errorf("routes[%d]: parse upstream: %w", i, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("routes[%d]: unsupported scheme %q", i, u.Scheme)
		}
		if u.Host == "" {
			return nil, fmt.Errorf("routes[%d]: upstream host is empty", i)
		}

		proto := strings.ToLower(strings.TrimSpace(rt.Proto))
		if proto == "" {
			proto = "http1"
		}
		switch proto {
		case "http1", "auto", "h2c":
		default:
			return nil, fmt.Errorf("routes[%d]: unknown proto %q", i, proto)
		}

		out.Routes = append(out.Routes, model.Route{
			Host: host, Prefix: pfx, URL: u, Proto: proto,
		})
	}

	// deterministic ordering: by host asc, then longer prefix first
	sort.SliceStable(out.Routes, func(i, j int) bool {
		if out.Routes[i].Host == out.Routes[j].Host {
			return len(out.Routes[i].Prefix) > len(out.Routes[j].Prefix)
		}
		return out.Routes[i].Host < out.Routes[j].Host
	})
	return out, nil
}
