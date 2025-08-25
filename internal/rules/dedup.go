package rules

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"ssh-noty/internal/model"
	"sync"
	"time"
)

type Deduper struct {
	ttl  time.Duration
	mu   sync.Mutex
	seen map[string]time.Time // key -> expiresAt
}

func NewDeduper(windowSeconds int) *Deduper {
	if windowSeconds <= 0 {
		windowSeconds = 30
	}
	return &Deduper{ttl: time.Duration(windowSeconds) * time.Second, seen: make(map[string]time.Time)}
}

// ShouldSend returns true if this event has not been seen recently (dedup window).
func (d *Deduper) ShouldSend(ev *model.Event) bool {
	k := eventKey(ev)
	now := time.Now()
	exp := now.Add(d.ttl)
	d.mu.Lock()
	defer d.mu.Unlock()
	// Cleanup lazily
	for key, when := range d.seen {
		if now.After(when) {
			delete(d.seen, key)
		}
	}
	if until, ok := d.seen[k]; ok && now.Before(until) {
		return false
	}
	d.seen[k] = exp
	return true
}

func eventKey(ev *model.Event) string {
	// Intentionally omit source port so repeated attempts with different ports dedup.
	base := fmt.Sprintf("%s|%s|%s|%s", ev.Type, ev.Username, ev.SourceIP, ev.Method)
	sum := sha1.Sum([]byte(base))
	return hex.EncodeToString(sum[:])
}
