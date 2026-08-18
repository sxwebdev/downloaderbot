package metrics

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/bits"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	hllPrecision       = 14
	hllRegisterCount   = 1 << hllPrecision
	activeUserDaySlots = 31
)

var processActiveUsers = mustNewActiveUsers()

type hllSketch struct {
	registers [hllRegisterCount]uint8
}

func (s *hllSketch) add(hash uint64) {
	index := hash >> (64 - hllPrecision)
	remaining := hash << hllPrecision
	rank := uint8(bits.LeadingZeros64(remaining) + 1)
	maxRank := uint8(64 - hllPrecision + 1)
	if rank > maxRank {
		rank = maxRank
	}
	if rank > s.registers[index] {
		s.registers[index] = rank
	}
}

func (s *hllSketch) merge(other *hllSketch) {
	for i, rank := range other.registers {
		if rank > s.registers[i] {
			s.registers[i] = rank
		}
	}
}

func (s *hllSketch) estimate() uint64 {
	registers := float64(hllRegisterCount)
	var sum float64
	zeros := 0
	for _, rank := range s.registers {
		sum += math.Ldexp(1, -int(rank))
		if rank == 0 {
			zeros++
		}
	}

	alpha := 0.7213 / (1 + 1.079/registers)
	estimate := alpha * registers * registers / sum
	if estimate <= 2.5*registers && zeros > 0 {
		estimate = registers * math.Log(registers/float64(zeros))
	}

	return uint64(math.Round(estimate))
}

type activeUserDay struct {
	day    int64
	valid  bool
	sketch hllSketch
}

// activeUsers stores only fixed-size HLL registers. Neither Telegram IDs nor
// their individual HMAC digests are retained or exported.
type activeUsers struct {
	mu     sync.RWMutex
	secret [sha256.Size]byte
	now    func() time.Time
	days   [activeUserDaySlots]activeUserDay
	desc   *prometheus.Desc
}

func mustNewActiveUsers() *activeUsers {
	var secret [sha256.Size]byte
	if _, err := rand.Read(secret[:]); err != nil {
		panic("initialize active-user metrics secret: " + err.Error())
	}
	return newActiveUsers(secret, time.Now)
}

func newActiveUsers(secret [sha256.Size]byte, now func() time.Time) *activeUsers {
	return &activeUsers{
		secret: secret,
		now:    now,
		desc: prometheus.NewDesc(
			"process_active_users_estimate",
			"Approximate unique Telegram senders observed by this process; resets on restart.",
			[]string{"window"},
			nil,
		),
	}
}

// ObserveActiveUser adds a Telegram sender to the process-local cardinality
// sketches. A zero ID is ignored because it means the update had no sender.
func ObserveActiveUser(userID int64) {
	processActiveUsers.observe(userID)
}

func (a *activeUsers) observe(userID int64) {
	if userID == 0 {
		return
	}

	mac := hmac.New(sha256.New, a.secret[:])
	mac.Write(strconv.AppendInt(nil, userID, 10))
	digest := mac.Sum(nil)
	hash := binary.BigEndian.Uint64(digest)
	day := utcDay(a.now())
	index := int(day % activeUserDaySlots)

	a.mu.Lock()
	defer a.mu.Unlock()
	slot := &a.days[index]
	if !slot.valid || slot.day != day {
		*slot = activeUserDay{day: day, valid: true}
	}
	slot.sketch.add(hash)
}

func (a *activeUsers) estimate(days int) uint64 {
	if days <= 0 || days > activeUserDaySlots {
		return 0
	}

	today := utcDay(a.now())
	oldest := today - int64(days) + 1
	var merged hllSketch

	a.mu.RLock()
	defer a.mu.RUnlock()
	for i := range a.days {
		slot := &a.days[i]
		if slot.valid && slot.day >= oldest && slot.day <= today {
			merged.merge(&slot.sketch)
		}
	}
	return merged.estimate()
}

func (a *activeUsers) Describe(ch chan<- *prometheus.Desc) {
	ch <- a.desc
}

func (a *activeUsers) Collect(ch chan<- prometheus.Metric) {
	for _, window := range []struct {
		label string
		days  int
	}{
		{label: "utc_day", days: 1},
		{label: "rolling_7d", days: 7},
		{label: "rolling_30d", days: 30},
	} {
		ch <- prometheus.MustNewConstMetric(
			a.desc,
			prometheus.GaugeValue,
			float64(a.estimate(window.days)),
			window.label,
		)
	}
}

func utcDay(t time.Time) int64 {
	return t.UTC().Unix() / int64((24 * time.Hour).Seconds())
}
