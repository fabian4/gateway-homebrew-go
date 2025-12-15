package metrics

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRegistry_IncRequest(t *testing.T) {
	r := NewRegistry()
	r.IncRequest("svc1", "route1", "GET", "200")
	r.IncRequest("svc1", "route1", "GET", "200")
	r.IncRequest("svc1", "route1", "POST", "500")

	var buf bytes.Buffer
	r.WritePrometheus(&buf)
	out := buf.String()

	if !strings.Contains(out, `requests_total{service="svc1",route="route1",method="GET",status="200"} 2`) {
		t.Errorf("missing GET 200 count 2:\n%s", out)
	}
	if !strings.Contains(out, `requests_total{service="svc1",route="route1",method="POST",status="500"} 1`) {
		t.Errorf("missing POST 500 count 1:\n%s", out)
	}
}

func TestRegistry_ActiveConns(t *testing.T) {
	r := NewRegistry()
	r.IncActiveConns("l1", "s1")
	r.IncActiveConns("l1", "s1")
	r.DecActiveConns("l1", "s1")

	var buf bytes.Buffer
	r.WritePrometheus(&buf)
	out := buf.String()

	if !strings.Contains(out, `active_connections{listener="l1",service="s1"} 1`) {
		t.Errorf("missing active conns 1:\n%s", out)
	}
}

func TestRegistry_ObserveLatency(t *testing.T) {
	r := NewRegistry()
	r.ObserveLatency("s1", "r1", 100*time.Millisecond) // 0.1s

	var buf bytes.Buffer
	r.WritePrometheus(&buf)
	out := buf.String()

	// Check bucket counts
	// 0.1 should fall into buckets >= 0.1
	if !strings.Contains(out, `upstream_latency_seconds_bucket{service="s1",route="r1",le="0.05"} 0`) {
		t.Errorf("bucket 0.05 should be 0:\n%s", out)
	}
	if !strings.Contains(out, `upstream_latency_seconds_bucket{service="s1",route="r1",le="0.1"} 1`) {
		t.Errorf("bucket 0.1 should be 1:\n%s", out)
	}
	if !strings.Contains(out, `upstream_latency_seconds_bucket{service="s1",route="r1",le="+Inf"} 1`) {
		t.Errorf("bucket +Inf should be 1:\n%s", out)
	}
	if !strings.Contains(out, `upstream_latency_seconds_sum{service="s1",route="r1"} 0.1`) {
		t.Errorf("sum should be 0.1:\n%s", out)
	}
	if !strings.Contains(out, `upstream_latency_seconds_count{service="s1",route="r1"} 1`) {
		t.Errorf("count should be 1:\n%s", out)
	}
}
