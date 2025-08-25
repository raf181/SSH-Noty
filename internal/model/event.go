package model

import "time"

type Event struct {
	Type           string
	Username       string
	SourceIP       string
	Port           int
	Method         string
	KeyFingerprint string
	Timestamp      time.Time
	Hostname       string
}
