package proxy

import (
	"sort"
	"strings"

	"github.com/fabian4/gateway-homebrew-go/internal/config"
)

type wildcardBucket struct {
	suffix string         // e.g. "example.com" for host "*.example.com"
	routes []config.Route // routes for that wildcard host, sorted by prefix desc
}

type Table struct {
	byHost   map[string][]config.Route // exact host -> routes sorted by prefix desc
	wildcard []wildcardBucket          // wildcard hosts ("*.example.com") ordered by longest suffix first
	any      []config.Route            // global wildcard routes (no host) -> prefix desc
}

func NewRouter(routes []config.Route) *Table {
	t := &Table{byHost: make(map[string][]config.Route)}

	// helper to collect wildcard hosts keyed by suffix
	wildBySuffix := make(map[string]*wildcardBucket)

	for _, r := range routes {
		h := strings.ToLower(strings.TrimSpace(r.Host))
		if h == "" {
			t.any = append(t.any, r)
			continue
		}
		if strings.HasPrefix(h, "*.") && len(h) > 2 {
			suffix := strings.TrimPrefix(h, "*.")
			b, ok := wildBySuffix[suffix]
			if !ok {
				b = &wildcardBucket{suffix: suffix}
				wildBySuffix[suffix] = b
			}
			b.routes = append(b.routes, r)
			continue
		}
		t.byHost[h] = append(t.byHost[h], r)
	}

	for h := range t.byHost {
		sort.SliceStable(t.byHost[h], func(i, j int) bool {
			return len(t.byHost[h][i].PathPrefix) > len(t.byHost[h][j].PathPrefix)
		})
	}

	for _, b := range wildBySuffix {
		sort.SliceStable(b.routes, func(i, j int) bool {
			return len(b.routes[i].PathPrefix) > len(b.routes[j].PathPrefix)
		})
		t.wildcard = append(t.wildcard, *b)
	}
	// more specific wildcard suffixes should be checked first
	sort.SliceStable(t.wildcard, func(i, j int) bool {
		return len(t.wildcard[i].suffix) > len(t.wildcard[j].suffix)
	})

	sort.SliceStable(t.any, func(i, j int) bool {
		return len(t.any[i].PathPrefix) > len(t.any[j].PathPrefix)
	})

	return t
}

func (t *Table) Match(host, path string) *config.Route {
	h := strings.ToLower(hostOnly(host))
	if r := match(t.byHost[h], path); r != nil {
		return r
	}

	// wildcard hosts: "*.example.com" style, only matching subdomains
	for _, b := range t.wildcard {
		if wildcardHostMatch(h, b.suffix) {
			if r := match(b.routes, path); r != nil {
				return r
			}
		}
	}

	return match(t.any, path)
}

func match(rs []config.Route, path string) *config.Route {
	for i := range rs {
		if pathPrefixMatch(path, rs[i].PathPrefix) {
			return &rs[i]
		}
	}
	return nil
}

// pathPrefixMatch ensures PathPrefix behaves like a path-segment prefix, not a raw string prefix.
// Examples:
//
//	prefix="/api"  matches "/api", "/api/", "/api/v1" but NOT "/apiary"
//	prefix="/api/" matches "/api/v1", "/api/foo" but NOT "/api"
//	prefix="/"     matches everything.
func pathPrefixMatch(path, prefix string) bool {
	if prefix == "/" {
		return true
	}
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	if len(path) == len(prefix) {
		return true
	}
	// At this point, path is strictly longer than prefix and shares the same bytes up to len(prefix).
	// We only consider it a match if the next character after the prefix boundary is a slash.
	// This prevents "/api" from matching "/apiary".
	return strings.HasSuffix(prefix, "/") || path[len(prefix)] == '/'
}

// wildcardHostMatch reports whether a concrete host is matched by a wildcard suffix.
// It implements "*.example.com" semantics:
//   - "api.example.com" matches suffix "example.com"
//   - "example.com" does NOT match suffix "example.com"
//   - "deep.api.example.com" also matches suffix "example.com"
func wildcardHostMatch(host, suffix string) bool {
	if host == "" || suffix == "" {
		return false
	}
	if len(host) <= len(suffix) {
		return false
	}
	if !strings.HasSuffix(host, suffix) {
		return false
	}
	// require a dot before the suffix to ensure we only match subdomains
	idx := len(host) - len(suffix) - 1
	return idx >= 0 && host[idx] == '.'
}

func hostOnly(h string) string {
	if i := strings.IndexByte(h, ':'); i >= 0 {
		return h[:i]
	}
	return h
}
