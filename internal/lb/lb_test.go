package lb

import (
	"net/url"
	"testing"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

func TestSmoothWRR(t *testing.T) {
	u1, _ := url.Parse("http://a")
	u2, _ := url.Parse("http://b")
	u3, _ := url.Parse("http://c")

	endpoints := []model.Endpoint{
		{URL: u1, Weight: 5},
		{URL: u2, Weight: 1},
		{URL: u3, Weight: 1},
	}

	lb := NewSmoothWRR(endpoints)

	// Total weight = 7
	// Expected sequence for smooth WRR (Nginx style):
	// A (5, 1, 1) -> current: 5, 1, 1 -> best A (5) -> current: -2, 1, 1
	// A (5, 1, 1) -> current: 3, 2, 2 -> best A (3) -> current: -4, 2, 2
	// B (5, 1, 1) -> current: 1, 3, 3 -> best B (3) -> current: 1, -4, 3
	// A (5, 1, 1) -> current: 6, -3, 4 -> best A (6) -> current: -1, -3, 4
	// C (5, 1, 1) -> current: 4, -2, 5 -> best C (5) -> current: 4, -2, -2
	// A (5, 1, 1) -> current: 9, -1, -1 -> best A (9) -> current: 2, -1, -1
	// A (5, 1, 1) -> current: 7, 0, 0 -> best A (7) -> current: 0, 0, 0

	expected := []string{"a", "a", "b", "a", "c", "a", "a"}

	for i, want := range expected {
		got := lb.Next()
		if got.Host != want {
			t.Errorf("step %d: got %s, want %s", i, got.Host, want)
		}
	}
}

func TestSmoothWRR_Single(t *testing.T) {
	u1, _ := url.Parse("http://a")
	endpoints := []model.Endpoint{{URL: u1, Weight: 1}}
	lb := NewSmoothWRR(endpoints)

	for i := 0; i < 10; i++ {
		if got := lb.Next(); got.Host != "a" {
			t.Errorf("got %s, want a", got.Host)
		}
	}
}
