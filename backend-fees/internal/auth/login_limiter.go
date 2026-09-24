package auth

import (
	"strings"
	"sync"
	"time"
)

// LoginLimiter throttles password guessing. It counts failed attempts in a
// fixed window per client IP + account and per client IP alone; once a limit
// is reached, further attempts are rejected until the window ends. Keying the
// account limit by IP keeps an attacker from locking out a user everywhere.
// State is in memory, which is enough for a single backend instance.
type LoginLimiter struct {
	mu            sync.Mutex
	maxPerAccount int
	maxPerIP      int
	window        time.Duration
	now           func() time.Time
	failures      map[string]*failureWindow
	lastPrune     time.Time
}

type failureWindow struct {
	count int
	start time.Time
}

// Default limits: 5 wrong passwords per account and IP, 20 per IP, per 15 minutes.
const (
	DefaultLoginMaxPerAccount = 5
	DefaultLoginMaxPerIP      = 20
	DefaultLoginWindow        = 15 * time.Minute
)

// NewLoginLimiter creates a limiter with the given limits per window.
func NewLoginLimiter(maxPerAccount, maxPerIP int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		maxPerAccount: maxPerAccount,
		maxPerIP:      maxPerIP,
		window:        window,
		now:           time.Now,
		failures:      make(map[string]*failureWindow),
	}
}

func accountKey(ip, account string) string {
	return "a|" + ip + "|" + strings.ToLower(strings.TrimSpace(account))
}

func ipKey(ip string) string {
	return "i|" + ip
}

// Blocked reports whether the next attempt must be rejected and, if so, how
// long until it is allowed again.
func (l *LoginLimiter) Blocked(ip, account string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	var retryAfter time.Duration
	for key, max := range map[string]int{accountKey(ip, account): l.maxPerAccount, ipKey(ip): l.maxPerIP} {
		w := l.active(key, now)
		if w != nil && w.count >= max {
			if wait := w.start.Add(l.window).Sub(now); wait > retryAfter {
				retryAfter = wait
			}
		}
	}
	return retryAfter > 0, retryAfter
}

// Fail records a failed attempt.
func (l *LoginLimiter) Fail(ip, account string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.prune(now)
	for _, key := range []string{accountKey(ip, account), ipKey(ip)} {
		if w := l.active(key, now); w != nil {
			w.count++
		} else {
			l.failures[key] = &failureWindow{count: 1, start: now}
		}
	}
}

// Succeed clears the account counter after a successful attempt. The IP
// counter stays, so valid logins cannot be used to reset a spraying budget.
func (l *LoginLimiter) Succeed(ip, account string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, accountKey(ip, account))
}

// active returns the window for key if it has not expired yet.
func (l *LoginLimiter) active(key string, now time.Time) *failureWindow {
	w, ok := l.failures[key]
	if !ok || now.Sub(w.start) >= l.window {
		return nil
	}
	return w
}

// prune drops expired windows at most once per window.
func (l *LoginLimiter) prune(now time.Time) {
	if now.Sub(l.lastPrune) < l.window {
		return
	}
	for key, w := range l.failures {
		if now.Sub(w.start) >= l.window {
			delete(l.failures, key)
		}
	}
	l.lastPrune = now
}
