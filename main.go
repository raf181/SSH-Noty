package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ssh-noty/internal/config"
	"ssh-noty/internal/enrich"
	"ssh-noty/internal/logging"
	"ssh-noty/internal/notify"
	"ssh-noty/internal/parser"
	"ssh-noty/internal/rules"
	"ssh-noty/internal/sources"
)

var (
	flagConfig  = flag.String("config", "/opt/ssh-noti/config.json", "Path to config.json")
	flagDaemon  = flag.Bool("daemon", false, "Run continuous monitoring daemon")
	flagBatch   = flag.Bool("batch", false, "Run batch summary and exit")
	flagTest    = flag.Bool("test", false, "Run self-check and send a test message")
	flagVersion = flag.Bool("version", false, "Print version and exit")
)

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Println(versionString())
		return
	}

	cfg, err := config.Load(*flagConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logging.Setup(cfg.Telemetry.LogLevel, cfg.Telemetry.LogFile)
	log := logging.L()

	if *flagTest {
		testRun(cfg)
		return
	}

	if *flagBatch {
		log.Info("batch mode not yet implemented; printing TODO and exiting")
		fmt.Println("TODO: batch summary mode is a planned feature.")
		os.Exit(0)
	}

	// Default to daemon unless flags say otherwise
	runDaemon(cfg)
}

func runDaemon(cfg *config.Config) {
	log := logging.L()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info("shutdown signal received")
		cancel()
	}()

	enricher := enrich.NewEnricher(cfg)
	slack := notify.NewSlack(cfg)
	prs := parser.NewParser()
	dedup := rules.NewDeduper(cfg.RateLimit.DedupWindowSeconds)

	src, err := sources.SelectSource(ctx, cfg)
	if err != nil {
		log.Error("failed to select source", "error", err)
		os.Exit(1)
	}
	records, err := src.Start(ctx)
	if err != nil {
		log.Error("failed to start source", "error", err)
		os.Exit(1)
	}

	log.Info("ssh-noti daemon started", "source", src.Name())
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("context cancelled; exiting")
			return
		case rec, ok := <-records:
			if !ok {
				log.Warn("records channel closed; exiting")
				return
			}
			if ev, ok := prs.Parse(rec); ok {
				enricher.Enrich(&ev)
				if !dedup.ShouldSend(&ev) {
					continue
				}
				if cfg.SlackWebhook == "" {
					log.Info("event", "type", ev.Type, "user", ev.Username, "ip", ev.SourceIP, "method", ev.Method)
					continue
				}
				if err := slack.SendEvent(ctx, &ev); err != nil {
					log.Warn("failed to send to slack", "error", err)
				}
			}
		case <-ticker.C:
			log.Debug("heartbeat")
		}
	}
}

func testRun(cfg *config.Config) {
	logging.Setup(cfg.Telemetry.LogLevel, cfg.Telemetry.LogFile)
	log := logging.L()
	slack := notify.NewSlack(cfg)
	msg := notify.TestMessage()
	if cfg.SlackWebhook == "" {
		log.Info("--test: no slack_webhook configured; printing test payload")
		fmt.Println(msg.Text)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := slack.Send(ctx, msg); err != nil {
		log.Error("test send failed", "error", err)
		os.Exit(1)
	}
	log.Info("test message sent successfully")
}

func versionString() string {
	return fmt.Sprintf("ssh-noti %s", Version)
}

// Version is overridden at build time via -ldflags "-X main.Version=..."
var Version = "dev"
