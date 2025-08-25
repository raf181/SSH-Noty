package parser

import (
	"testing"
	"time"
)

func rr(line string) RawRecord {
	return RawRecord{Line: line, Timestamp: time.Now(), Hostname: "test-host"}
}

func TestParse_PreauthWithUsername(t *testing.T) {
	p := NewParser()
	// Connection closed/reset with authenticating user
	lines := []string{
		"Connection closed by authenticating user alice 192.0.2.4 port 12345 [preauth]",
		"Connection reset by authenticating user alice 192.0.2.4 port 12345 [preauth]",
		"Disconnected from authenticating user alice 192.0.2.4 port 12345 [preauth]",
		// invalid user variant
		"Connection closed by invalid user alice 192.0.2.4 port 12345 [preauth]",
	}
	for _, line := range lines {
		ev, ok := p.Parse(rr(line))
		if !ok {
			t.Fatalf("expected parse ok for line: %q", line)
		}
		if ev.Type != "login_failure" || ev.Method != "preauth" {
			t.Fatalf("unexpected event: %+v", ev)
		}
		if ev.Username != "alice" || ev.SourceIP != "192.0.2.4" || ev.Port != 12345 {
			t.Fatalf("unexpected fields: %+v", ev)
		}
	}
}

func TestParse_PreauthIPOnly(t *testing.T) {
	p := NewParser()
	lines := []string{
		"Disconnected from 203.0.113.5 port 2200 [preauth]",
		"Connection reset by 2001:db8::1 port 2222 [preauth]",
		"Connection closed by 203.0.113.6 port 2223 [preauth]",
	}
	for _, line := range lines {
		ev, ok := p.Parse(rr(line))
		if !ok {
			t.Fatalf("expected parse ok for line: %q", line)
		}
		if ev.Type != "login_failure" || ev.Method != "preauth" {
			t.Fatalf("unexpected event: %+v", ev)
		}
		if ev.Username != "" {
			t.Fatalf("expected empty username, got: %q", ev.Username)
		}
		if ev.SourceIP == "" || ev.Port == 0 {
			t.Fatalf("expected source ip and port to be set: %+v", ev)
		}
	}
}

func TestParse_FailedMethods(t *testing.T) {
	p := NewParser()
	cases := []struct {
		line   string
		method string
	}{
		{"Failed publickey for alice from 203.0.113.10 port 4444", "publickey"},
		{"Failed keyboard-interactive/pam for alice from 203.0.113.10 port 4445", "keyboard-interactive"},
		{"Failed keyboard-interactive for alice from 203.0.113.10 port 4446", "keyboard-interactive"},
		{"Failed none for alice from 203.0.113.10 port 4447", "none"},
	}
	for _, tc := range cases {
		ev, ok := p.Parse(rr(tc.line))
		if !ok {
			t.Fatalf("expected parse ok for line: %q", tc.line)
		}
		if ev.Type != "login_failure" || ev.Method != tc.method {
			t.Fatalf("unexpected event: %+v", ev)
		}
		if ev.Username != "alice" || ev.SourceIP == "" || ev.Port == 0 {
			t.Fatalf("unexpected fields: %+v", ev)
		}
	}
}

func TestParse_PamUnixFailure(t *testing.T) {
	p := NewParser()
	line := "pam_unix(sshd:auth): authentication failure; logname= uid=0 euid=0 tty=ssh ruser= rhost=203.0.113.8 user=root"
	ev, ok := p.Parse(rr(line))
	if !ok {
		t.Fatalf("expected parse ok for pam_unix line")
	}
	if ev.Type != "login_failure" || ev.Method != "pam" || ev.Username != "root" || ev.SourceIP != "203.0.113.8" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParse_MaxAttempts(t *testing.T) {
	p := NewParser()
	lines := []string{
		"error: maximum authentication attempts exceeded for invalid user bob from 203.0.113.9 port 2222 ssh2 [preauth]",
		"error: maximum authentication attempts exceeded for alice from 203.0.113.9 port 2222 ssh2 [preauth]",
	}
	wantUser := []string{"bob", "alice"}
	for i, line := range lines {
		ev, ok := p.Parse(rr(line))
		if !ok {
			t.Fatalf("expected parse ok for line: %q", line)
		}
		if ev.Type != "login_failure" || ev.Method != "max-attempts" {
			t.Fatalf("unexpected event: %+v", ev)
		}
		if ev.Username != wantUser[i] || ev.SourceIP != "203.0.113.9" || ev.Port != 2222 {
			t.Fatalf("unexpected fields: %+v", ev)
		}
	}
}
