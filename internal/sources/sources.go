package sources

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"ssh-noty/internal/config"
	"ssh-noty/internal/parser"
)

type Source interface {
	Start(ctx context.Context) (<-chan parser.RawRecord, error)
	Name() string
}

func SelectSource(ctx context.Context, cfg *config.Config) (Source, error) {
	prefer := cfg.Sources.Prefer
	if prefer == "auto" || prefer == "journald" {
		if _, err := exec.LookPath("journalctl"); err == nil {
			return &JournalctlFollower{Units: cfg.Sources.SystemdUnits}, nil
		}
		if prefer == "journald" {
			return nil, errors.New("journalctl not found")
		}
	}
	// fallback to file tail
	paths := cfg.Sources.FilePaths
	if len(paths) == 0 {
		paths = []string{"/var/log/auth.log", "/var/log/secure", "/var/log/messages"}
	}
	return &FileFollower{Paths: paths}, nil
}

// JournalctlFollower streams journal entries for sshd units and emits RawRecord lines from MESSAGE
type JournalctlFollower struct {
	Units []string
}

func (j *JournalctlFollower) Name() string { return "journalctl" }

func (j *JournalctlFollower) Start(ctx context.Context) (<-chan parser.RawRecord, error) {
	args := []string{"-o", "json", "-f"}
	for _, u := range j.Units {
		args = append(args, "-u", u)
	}
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan parser.RawRecord, 200)
	go func() {
		defer close(ch)
		defer cmd.Wait()
		scanner := bufio.NewScanner(stdout)
		// increase buffer
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		hostname, _ := os.Hostname()
		for scanner.Scan() {
			var m map[string]any
			line := scanner.Text()
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				continue
			}
			msg, _ := m["MESSAGE"].(string)
			ts := time.Now()
			if v, ok := m["__REALTIME_TIMESTAMP"].(string); ok {
				// not converting microseconds here; keep now
				_ = v
			}
			if strings.TrimSpace(msg) == "" {
				continue
			}
			ch <- parser.RawRecord{Line: msg, Timestamp: ts, Hostname: hostname}
		}
	}()
	return ch, nil
}

// FileFollower is a minimal file tailing fallback (no rotation handling yet)
type FileFollower struct {
	Paths []string
}

func (f *FileFollower) Name() string { return "file" }

func (f *FileFollower) Start(ctx context.Context) (<-chan parser.RawRecord, error) {
	ch := make(chan parser.RawRecord)
	go func() {
		defer close(ch)
		// naive: follow only the first existing file
		var chosen string
		for _, p := range f.Paths {
			if _, err := os.Stat(p); err == nil {
				chosen = p
				break
			}
		}
		if chosen == "" {
			return
		}
		file, err := os.Open(chosen)
		if err != nil {
			return
		}
		defer file.Close()
		// seek to end
		file.Seek(0, 2)
		reader := bufio.NewReader(file)
		hostname, _ := os.Hostname()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					ch <- parser.RawRecord{Line: strings.TrimRight(line, "\r\n"), Timestamp: time.Now(), Hostname: hostname}
				}
				if err != nil {
					time.Sleep(500 * time.Millisecond)
				}
			}
		}
	}()
	return ch, nil
}
