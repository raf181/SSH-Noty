package parser

import (
	"regexp"
	"ssh-noty/internal/model"
	"strings"
	"time"
)

type RawRecord struct {
	Line      string
	Timestamp time.Time
	Hostname  string
}

// Event is now moved to internal/model

type Parser struct {
	reSuccess       *regexp.Regexp
	reFailure       *regexp.Regexp
	reInvalid       *regexp.Regexp
	reConnClosed    *regexp.Regexp
	reMaxAttempts   *regexp.Regexp
	rePamFailure    *regexp.Regexp
	rePreauthIPOnly *regexp.Regexp
}

func NewParser() *Parser {
	return &Parser{
		reSuccess: regexp.MustCompile(`^Accepted (password|publickey|keyboard-interactive) for (\S+) from ([\da-fA-F:\.]+) port (\d+)`),
		// Capture method for failures: password, publickey, keyboard-interactive (optionally /pam), or none
		reFailure: regexp.MustCompile(`^Failed (password|publickey|keyboard-interactive(?:/pam)?|none) for (?:invalid user )?(\S+) from ([\da-fA-F:\.]+) port (\d+)`),
		reInvalid: regexp.MustCompile(`^Invalid user (\S+) from ([\da-fA-F:\.]+)`),
		// Disconnected/closed/reset before auth completes (treat as failure)
		reConnClosed: regexp.MustCompile(`^(?:Disconnected from|Connection (?:closed|reset) by) (?:invalid user )?(?:authenticating user )?(\S+) ([\da-fA-F:\.]+) port (\d+) \[preauth\]`),
		// Maximum authentication attempts exceeded
		reMaxAttempts: regexp.MustCompile(`^error: maximum authentication attempts exceeded for (?:invalid user )?(\S+) from ([\da-fA-F:\.]+) port (\d+) ssh2 \[preauth\]`),
		// PAM failure line
		rePamFailure: regexp.MustCompile(`^pam_unix\(sshd:auth\): authentication failure;.*rhost=([\da-fA-F:\.]+) user=(\S+)`),
		// Preauth line with only IP (no username)
		rePreauthIPOnly: regexp.MustCompile(`^(?:Disconnected from|Connection (?:closed|reset) by) ([\da-fA-F:\.]+) port (\d+) \[preauth\]`),
	}
}

func (p *Parser) Parse(rr RawRecord) (model.Event, bool) {
	line := strings.TrimSpace(rr.Line)
	if m := p.reSuccess.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "login_success", Method: m[1], Username: m[2], SourceIP: m[3], Port: atoi(m[4]), Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	if m := p.reFailure.FindStringSubmatch(line); m != nil {
		method := m[1]
		// Normalize keyboard-interactive/pam to keyboard-interactive
		if strings.HasPrefix(method, "keyboard-interactive") {
			method = "keyboard-interactive"
		}
		return model.Event{Type: "login_failure", Method: method, Username: m[2], SourceIP: m[3], Port: atoi(m[4]), Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	if m := p.reInvalid.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "invalid_user", Username: m[1], SourceIP: m[2], Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	if m := p.reConnClosed.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "login_failure", Method: "preauth", Username: m[1], SourceIP: m[2], Port: atoi(m[3]), Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	if m := p.rePreauthIPOnly.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "login_failure", Method: "preauth", Username: "", SourceIP: m[1], Port: atoi(m[2]), Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	if m := p.reMaxAttempts.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "login_failure", Method: "max-attempts", Username: m[1], SourceIP: m[2], Port: atoi(m[3]), Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	if m := p.rePamFailure.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "login_failure", Method: "pam", Username: m[2], SourceIP: m[1], Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	return model.Event{}, false
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
