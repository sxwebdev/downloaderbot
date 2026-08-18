package metrics

import (
	"crypto/sha256"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestActiveUsersDeduplicatesAcrossWindows(t *testing.T) {
	current := time.Date(2026, time.August, 1, 12, 0, 0, 0, time.FixedZone("test", 3*60*60))
	users := newActiveUsers(testSecret(), func() time.Time { return current })

	users.observe(101)
	users.observe(101)
	users.observe(202)
	if got := users.estimate(1); got != 2 {
		t.Fatalf("day 1 estimate = %d, want 2", got)
	}

	current = current.Add(24 * time.Hour)
	users.observe(202)
	users.observe(303)
	if got := users.estimate(1); got != 2 {
		t.Errorf("day 2 estimate = %d, want 2", got)
	}
	if got := users.estimate(7); got != 3 {
		t.Errorf("rolling 7d estimate = %d, want 3", got)
	}
	if got := users.estimate(30); got != 3 {
		t.Errorf("rolling 30d estimate = %d, want 3", got)
	}

	// Reusing a ring slot must discard users older than the 30-day horizon.
	current = current.Add(30 * 24 * time.Hour)
	users.observe(404)
	if got := users.estimate(30); got != 1 {
		t.Errorf("estimate after ring reuse = %d, want 1", got)
	}
}

func TestActiveUsersIgnoresMissingSender(t *testing.T) {
	users := newActiveUsers(testSecret(), func() time.Time {
		return time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	})
	users.observe(0)
	if got := users.estimate(1); got != 0 {
		t.Fatalf("estimate = %d, want 0", got)
	}
	if got := users.estimate(0); got != 0 {
		t.Errorf("zero-day estimate = %d, want 0", got)
	}
	if got := users.estimate(activeUserDaySlots + 1); got != 0 {
		t.Errorf("out-of-range estimate = %d, want 0", got)
	}
}

func TestObserveActiveUserUsesProcessCollector(t *testing.T) {
	original := processActiveUsers
	processActiveUsers = newActiveUsers(testSecret(), func() time.Time {
		return time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	})
	t.Cleanup(func() { processActiveUsers = original })

	ObserveActiveUser(42)
	ObserveActiveUser(42)
	if got := processActiveUsers.estimate(1); got != 1 {
		t.Fatalf("estimate = %d, want one deduplicated user", got)
	}
}

func TestHLLMaximumRank(t *testing.T) {
	var sketch hllSketch
	sketch.add(0)
	want := uint8(64 - hllPrecision + 1)
	if got := sketch.registers[0]; got != want {
		t.Fatalf("rank = %d, want capped maximum %d", got, want)
	}
}

func TestActiveUsersAccuracyAndConcurrency(t *testing.T) {
	const uniqueUsers = 100_000
	users := newActiveUsers(testSecret(), func() time.Time {
		return time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	})

	const workers = 10
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := worker * (uniqueUsers / workers)
			end := start + uniqueUsers/workers
			for id := start + 1; id <= end; id++ {
				users.observe(int64(id))
			}
		}()
	}
	wg.Wait()

	got := users.estimate(1)
	relativeError := abs(float64(got)-uniqueUsers) / uniqueUsers
	if relativeError > 0.02 {
		t.Fatalf("estimate = %d for %d users, relative error %.3f exceeds 2%%", got, uniqueUsers, relativeError)
	}
}

func TestActiveUsersCollectorContract(t *testing.T) {
	current := time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC)
	users := newActiveUsers(testSecret(), func() time.Time { return current })
	users.observe(1)
	users.observe(2)

	want := `
# HELP process_active_users_estimate Approximate unique Telegram senders observed by this process; resets on restart.
# TYPE process_active_users_estimate gauge
process_active_users_estimate{window="rolling_30d"} 2
process_active_users_estimate{window="rolling_7d"} 2
process_active_users_estimate{window="utc_day"} 2
`
	if err := testutil.CollectAndCompare(users, strings.NewReader(want), "process_active_users_estimate"); err != nil {
		t.Fatalf("collector output: %v", err)
	}
}

func testSecret() [sha256.Size]byte {
	return sha256.Sum256([]byte("deterministic test-only HLL secret"))
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
