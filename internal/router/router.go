package router

import (
	"sort"
	"strings"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

type Table struct {
	byHost map[string][]model.Route // exact host -> routes sorted by prefix desc
	any    []model.Route            // wildcard routes -> prefix desc
}

func New(routes []model.Route) *Table {
	t := &Table{byHost: make(map[string][]model.Route)}
	for _, r := range routes {
		if len(r.Hosts) == 0 {
			t.any = append(t.any, r)
			continue
		}
		for _, h := range r.Hosts {
			h = strings.ToLower(h)
			t.byHost[h] = append(t.byHost[h], r)
		}
	}
	for h := range t.byHost {
		sort.SliceStable(t.byHost[h], func(i, j int) bool {
			return len(t.byHost[h][i].PathPrefix) > len(t.byHost[h][j].PathPrefix)
		})
	}
	sort.SliceStable(t.any, func(i, j int) bool {
		return len(t.any[i].PathPrefix) > len(t.any[j].PathPrefix)
	})
	return t
}

func (t *Table) Match(host, path string) *model.Route {
	h := strings.ToLower(hostOnly(host))
	if r := match(t.byHost[h], path); r != nil {
		return r
	}
	return match(t.any, path)
}

func match(rs []model.Route, path string) *model.Route {
	for i := range rs {
		if strings.HasPrefix(path, rs[i].PathPrefix) {
			return &rs[i]
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
