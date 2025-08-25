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
	reSuccess *regexp.Regexp
	reFailure *regexp.Regexp
	reInvalid *regexp.Regexp
}

func NewParser() *Parser {
	return &Parser{
		reSuccess: regexp.MustCompile(`^Accepted (password|publickey|keyboard-interactive) for (\S+) from ([\da-fA-F:\.]+) port (\d+)`),
		reFailure: regexp.MustCompile(`^Failed password for (?:invalid user )?(\S+) from ([\da-fA-F:\.]+) port (\d+)`),
		reInvalid: regexp.MustCompile(`^Invalid user (\S+) from ([\da-fA-F:\.]+)`),
	}
}

func (p *Parser) Parse(rr RawRecord) (model.Event, bool) {
	line := strings.TrimSpace(rr.Line)
	if m := p.reSuccess.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "login_success", Method: m[1], Username: m[2], SourceIP: m[3], Port: atoi(m[4]), Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	if m := p.reFailure.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "login_failure", Method: "password", Username: m[1], SourceIP: m[2], Port: atoi(m[3]), Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
	}
	if m := p.reInvalid.FindStringSubmatch(line); m != nil {
		return model.Event{Type: "invalid_user", Username: m[1], SourceIP: m[2], Timestamp: rr.Timestamp, Hostname: rr.Hostname}, true
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
