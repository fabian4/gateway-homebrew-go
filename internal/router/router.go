package router

import (
	"sort"
	"strings"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

type Table struct {
	byHost map[string][]model.Route // exact host -> sorted by prefix length desc
	any    []model.Route            // wildcard host -> sorted by prefix length desc
}

func New(routes []model.Route) *Table {
	t := &Table{byHost: make(map[string][]model.Route)}
	for _, r := range routes {
		if r.Host == "" {
			t.any = append(t.any, r)
		} else {
			t.byHost[r.Host] = append(t.byHost[r.Host], r)
		}
	}
	for h := range t.byHost {
		sort.SliceStable(t.byHost[h], func(i, j int) bool {
			return len(t.byHost[h][i].Prefix) > len(t.byHost[h][j].Prefix)
		})
	}
	sort.SliceStable(t.any, func(i, j int) bool {
		return len(t.any[i].Prefix) > len(t.any[j].Prefix)
	})
	return t
}

// Match returns the matched route (not a copy), or nil if no match.
func (t *Table) Match(host, path string) *model.Route {
	h := strings.ToLower(hostOnly(host))
	if r := matchPrefixes(t.byHost[h], path); r != nil {
		return r
	}
	return matchPrefixes(t.any, path)
}

func matchPrefixes(routes []model.Route, path string) *model.Route {
	for i := range routes {
		if strings.HasPrefix(path, routes[i].Prefix) {
			return &routes[i]
		}
	}
	return nil
}

func hostOnly(h string) string {
	if i := strings.IndexByte(h, ':'); i >= 0 {
		return h[:i]
	}
	return h
}
