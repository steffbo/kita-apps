package auth

import (
	"testing"
	"time"
)

func newTestLimiter(now *time.Time) *LoginLimiter {
	l := NewLoginLimiter(3, 5, 10*time.Minute)
	l.now = func() time.Time { return *now }
	return l
}

func TestLoginLimiter_BlocksAccountAfterMaxFailures(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	l := newTestLimiter(&now)

	for i := 0; i < 3; i++ {
		if blocked, _ := l.Blocked("1.1.1.1", "a@example.com"); blocked {
			t.Fatalf("blocked after %d failures", i)
		}
		l.Fail("1.1.1.1", "a@example.com")
	}
	blocked, retry := l.Blocked("1.1.1.1", "A@Example.com ")
	if !blocked || retry != 10*time.Minute {
		t.Fatalf("blocked=%v retry=%v, want true/10m (email is case-insensitive)", blocked, retry)
	}

	// Other IPs and other accounts are unaffected.
	if blocked, _ := l.Blocked("2.2.2.2", "a@example.com"); blocked {
		t.Error("other IP blocked")
	}
	if blocked, _ := l.Blocked("1.1.1.1", "b@example.com"); blocked {
		t.Error("other account blocked")
	}

	now = now.Add(10 * time.Minute)
	if blocked, _ := l.Blocked("1.1.1.1", "a@example.com"); blocked {
		t.Error("still blocked after window")
	}
}

func TestLoginLimiter_BlocksIPAcrossAccounts(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	l := newTestLimiter(&now)

	for i := 0; i < 5; i++ {
		l.Fail("1.1.1.1", "user"+string(rune('a'+i))+"@example.com")
	}
	if blocked, _ := l.Blocked("1.1.1.1", "fresh@example.com"); !blocked {
		t.Error("IP not blocked after spraying")
	}
}

func TestLoginLimiter_SuccessResetsAccountOnly(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	l := newTestLimiter(&now)

	l.Fail("1.1.1.1", "a@example.com")
	l.Fail("1.1.1.1", "a@example.com")
	l.Succeed("1.1.1.1", "a@example.com")
	l.Fail("1.1.1.1", "a@example.com")
	l.Fail("1.1.1.1", "a@example.com")
	if blocked, _ := l.Blocked("1.1.1.1", "a@example.com"); blocked {
		t.Error("account counter not reset by success")
	}
	// The IP counter kept all 4 failures; one more reaches the IP limit of 5.
	l.Fail("1.1.1.1", "b@example.com")
	if blocked, _ := l.Blocked("1.1.1.1", "c@example.com"); !blocked {
		t.Error("IP counter was reset by success")
	}
}
