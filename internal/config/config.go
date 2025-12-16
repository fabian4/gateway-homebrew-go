package config

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type rawConfig struct {
	EntryPoint []struct {
		Name    string `yaml:"name"`
		Address string `yaml:"address"`
		Service string `yaml:"service"`
	} `yaml:"entrypoint"`
	Services []struct {
		Name      string `yaml:"name"`
		Proto     string `yaml:"proto"`
		Endpoints []any  `yaml:"endpoints"`
		TLS       struct {
			InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
			CAFile             string `yaml:"ca_file"`
			CertFile           string `yaml:"cert_file"`
			KeyFile            string `yaml:"key_file"`
		} `yaml:"tls"`
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
		Read          string `yaml:"read"`
		Write         string `yaml:"write"`
		Upstream      string `yaml:"upstream"`
		TCPIdle       string `yaml:"tcp_idle"`
		TCPConnection string `yaml:"tcp_connection"`
	} `yaml:"timeouts"`
	TLS struct {
		Enabled      bool `yaml:"enabled"`
		Certificates []struct {
			CertFile string `yaml:"cert_file"`
			KeyFile  string `yaml:"key_file"`
		} `yaml:"certificates"`
	} `yaml:"tls"`
	Metrics struct {
		Address string `yaml:"address"`
	} `yaml:"metrics"`
	AccessLog struct {
		Fields   []string `yaml:"fields"`
		Sampling *float64 `yaml:"sampling"`
	} `yaml:"access_log"`
	Transport struct {
		MaxIdleConns        int    `yaml:"max_idle_conns"`
		MaxIdleConnsPerHost int    `yaml:"max_idle_conns_per_host"`
		IdleConnTimeout     string `yaml:"idle_conn_timeout"`
		DialTimeout         string `yaml:"dial_timeout"`
		DialKeepAlive       string `yaml:"dial_keep_alive"`
	} `yaml:"transport"`
	RefreshInterval string `yaml:"refresh_interval"`
}

type Config struct {
	Listen          string
	RefreshInterval time.Duration
	Listeners       []Listener
	Services        map[string]Service
	Routes          []Route
	Timeouts        Timeouts
	TLS             TLSConfig
	Metrics         MetricsConfig
	AccessLog       AccessLogConfig
	Transport       TransportConfig
}

type TransportConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	DialTimeout         time.Duration
	DialKeepAlive       time.Duration
}

type MetricsConfig struct {
	Address string
}

type AccessLogConfig struct {
	Fields   []string
	Sampling float64
}

type TLSConfig struct {
	Enabled      bool
	Certificates []Certificate
}

type Certificate struct {
	CertFile string
	KeyFile  string
}

type Timeouts struct {
	Read          time.Duration
	Write         time.Duration
	Upstream      time.Duration
	TCPIdle       time.Duration
	TCPConnection time.Duration
}

const (
	DefaultReadTimeout    = 15 * time.Second
	DefaultWriteTimeout   = 30 * time.Second
	DefaultTCPIdleTimeout = 5 * time.Minute
)

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
	var listeners []Listener
	if len(rc.EntryPoint) > 0 {
		for _, ep := range rc.EntryPoint {
			addr := strings.TrimSpace(ep.Address)
			if addr == "" {
				addr = ":8080"
			}
			listeners = append(listeners, Listener{
				Name:    strings.TrimSpace(ep.Name),
				Address: addr,
				Service: strings.TrimSpace(ep.Service),
			})
		}
	} else {
		listeners = append(listeners, Listener{
			Name:    "default",
			Address: ":8080",
		})
	}

	// backward compatibility for Listen field (use first listener)
	listen := listeners[0].Address

	// services
	svcs := make(map[string]Service)
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
		case "http1", "auto", "h2c", "tcp":
		default:
			return nil, fmt.Errorf("services[%d]: unknown proto %q", i, proto)
		}
		if len(s.Endpoints) == 0 {
			return nil, fmt.Errorf("services[%d]: endpoints is empty", i)
		}
		var eps []Endpoint
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
			if (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "tcp") || u.Host == "" {
				return nil, fmt.Errorf("services[%d].endpoints[%d]: must be http(s) or tcp URL with host", i, j)
			}
			eps = append(eps, Endpoint{URL: u, Weight: weight})
		}
		if _, dup := svcs[name]; dup {
			return nil, fmt.Errorf("services: duplicate name %q", name)
		}
		var upstreamTLS *UpstreamTLS
		if s.TLS.InsecureSkipVerify || s.TLS.CAFile != "" || s.TLS.CertFile != "" || s.TLS.KeyFile != "" {
			upstreamTLS = &UpstreamTLS{
				InsecureSkipVerify: s.TLS.InsecureSkipVerify,
				CAFile:             s.TLS.CAFile,
				CertFile:           s.TLS.CertFile,
				KeyFile:            s.TLS.KeyFile,
			}
		}
		svcs[name] = Service{
			Name:      name,
			Proto:     proto,
			Endpoints: eps,
			TLS:       upstreamTLS,
		}
	}
	if len(svcs) == 0 {
		return nil, fmt.Errorf("services: at least one is required")
	}

	// routes
	var routes []Route
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
		rt := Route{
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
	timeouts.Read = DefaultReadTimeout
	if rc.Timeouts.Read != "" {
		d, err := time.ParseDuration(rc.Timeouts.Read)
		if err != nil {
			return nil, fmt.Errorf("timeouts.read: %v", err)
		}
		timeouts.Read = d
	}
	timeouts.Write = DefaultWriteTimeout
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

	timeouts.TCPIdle = DefaultTCPIdleTimeout
	if rc.Timeouts.TCPIdle != "" {
		d, err := time.ParseDuration(rc.Timeouts.TCPIdle)
		if err != nil {
			return nil, fmt.Errorf("timeouts.tcp_idle: %v", err)
		}
		timeouts.TCPIdle = d
	}
	if rc.Timeouts.TCPConnection != "" {
		d, err := time.ParseDuration(rc.Timeouts.TCPConnection)
		if err != nil {
			return nil, fmt.Errorf("timeouts.tcp_connection: %v", err)
		}
		timeouts.TCPConnection = d
	}

	// tls
	var tlsConfig TLSConfig
	tlsConfig.Enabled = rc.TLS.Enabled
	if tlsConfig.Enabled {
		for i, c := range rc.TLS.Certificates {
			if c.CertFile == "" || c.KeyFile == "" {
				return nil, fmt.Errorf("tls.certificates[%d]: cert_file and key_file are required", i)
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, Certificate{
				CertFile: c.CertFile,
				KeyFile:  c.KeyFile,
			})
		}
		if len(tlsConfig.Certificates) == 0 {
			return nil, fmt.Errorf("tls.enabled is true but no certificates provided")
		}
	}

	// access log
	var accessLog AccessLogConfig
	accessLog.Fields = rc.AccessLog.Fields
	if rc.AccessLog.Sampling != nil {
		accessLog.Sampling = *rc.AccessLog.Sampling
	} else {
		accessLog.Sampling = 1.0 // default 100%
	}

	// transport
	var transport TransportConfig
	transport.MaxIdleConns = rc.Transport.MaxIdleConns
	if transport.MaxIdleConns == 0 {
		transport.MaxIdleConns = 512 // default
	}
	transport.MaxIdleConnsPerHost = rc.Transport.MaxIdleConnsPerHost
	if transport.MaxIdleConnsPerHost == 0 {
		transport.MaxIdleConnsPerHost = 128 // default
	}

	if rc.Transport.IdleConnTimeout != "" {
		d, err := time.ParseDuration(rc.Transport.IdleConnTimeout)
		if err != nil {
			return nil, fmt.Errorf("transport.idle_conn_timeout: %v", err)
		}
		transport.IdleConnTimeout = d
	} else {
		transport.IdleConnTimeout = 90 * time.Second
	}

	if rc.Transport.DialTimeout != "" {
		d, err := time.ParseDuration(rc.Transport.DialTimeout)
		if err != nil {
			return nil, fmt.Errorf("transport.dial_timeout: %v", err)
		}
		transport.DialTimeout = d
	} else {
		transport.DialTimeout = 5 * time.Second
	}

	if rc.Transport.DialKeepAlive != "" {
		d, err := time.ParseDuration(rc.Transport.DialKeepAlive)
		if err != nil {
			return nil, fmt.Errorf("transport.dial_keep_alive: %v", err)
		}
		transport.DialKeepAlive = d
	} else {
		transport.DialKeepAlive = 60 * time.Second
	}

	var refreshInterval time.Duration
	if rc.RefreshInterval != "" {
		d, err := time.ParseDuration(rc.RefreshInterval)
		if err != nil {
			return nil, fmt.Errorf("refresh_interval: %v", err)
		}
		refreshInterval = d
	} else {
		refreshInterval = 5 * time.Second
	}

	return &Config{
		Listen:          listen,
		RefreshInterval: refreshInterval,
		Listeners:       listeners,
		Services:        svcs,
		Routes:          routes,
		Timeouts:        timeouts,
		TLS:             tlsConfig,
		Metrics:         MetricsConfig{Address: rc.Metrics.Address},
		AccessLog:       accessLog,
		Transport:       transport,
	}, nil
}
