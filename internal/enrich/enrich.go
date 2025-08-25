package enrich

import (
	"os"
	"ssh-noty/internal/config"
	"ssh-noty/internal/model"
)

type Enricher struct{ cfg *config.Config }

func NewEnricher(cfg *config.Config) *Enricher { return &Enricher{cfg: cfg} }

func (e *Enricher) Enrich(ev *model.Event) {
	if ev.Hostname == "" {
		if h, _ := os.Hostname(); h != "" {
			ev.Hostname = h
		}
	}
}
