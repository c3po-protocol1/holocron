package claudecode

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
)

// DefaultPollInterval is the default process polling interval.
const DefaultPollInterval = 2 * time.Second

// Provider monitors Claude Code sessions.
type Provider struct {
	sessionDir   string
	pollInterval time.Duration

	bus    collector.EventBus
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu       sync.Mutex
	watcher  *fsnotify.Watcher
	tailing  map[string]context.CancelFunc // sessionID → cancel func
}

// New creates a new Claude Code provider.
func New(sessionDir string, pollInterval time.Duration) *Provider {
	if pollInterval == 0 {
		pollInterval = DefaultPollInterval
	}
	return &Provider{
		sessionDir:   sessionDir,
		pollInterval: pollInterval,
		tailing:      make(map[string]context.CancelFunc),
	}
}

func (p *Provider) Name() string { return sourceName }

func (p *Provider) Start(ctx context.Context, bus collector.EventBus) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.bus = bus

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cancel()
		return err
	}
	p.watcher = watcher

	// Initial scan
	sessions, err := ScanSessions(p.sessionDir)
	if err != nil {
		slog.Warn("initial scan failed", "error", err)
	}

	for _, s := range sessions {
		p.emitSessionStart(s)
		p.startTailing(ctx, s)
	}

	// Watch for new session directories
	if err := watcher.Add(p.sessionDir); err != nil {
		slog.Warn("cannot watch session dir", "error", err)
	}

	// Also watch existing subdirectories for new .jsonl files
	entries, _ := os.ReadDir(p.sessionDir)
	for _, e := range entries {
		if e.IsDir() {
			_ = watcher.Add(filepath.Join(p.sessionDir, e.Name()))
		}
	}

	// Directory watcher goroutine
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.watchDirectories(ctx)
	}()

	// Process poller goroutine
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.pollProcesses(ctx)
	}()

	return nil
}

func (p *Provider) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}

	p.mu.Lock()
	for sid, cancelFn := range p.tailing {
		cancelFn()
		delete(p.tailing, sid)
	}
	p.mu.Unlock()

	if p.watcher != nil {
		p.watcher.Close()
	}

	p.wg.Wait()
	return nil
}

func (p *Provider) emitSessionStart(s SessionFile) {
	p.bus.Publish(collector.MonitorEvent{
		ID:        uuid.New().String(),
		Source:    sourceName,
		SessionID: s.SessionID,
		Workspace: s.Workspace,
		Timestamp: time.Now().UnixMilli(),
		Event:     collector.EventSessionStart,
		Status:    collector.StatusIdle,
	})
}

func (p *Provider) startTailing(ctx context.Context, s SessionFile) {
	p.mu.Lock()
	if _, ok := p.tailing[s.SessionID]; ok {
		p.mu.Unlock()
		return // already tailing
	}
	tailCtx, tailCancel := context.WithCancel(ctx)
	p.tailing[s.SessionID] = tailCancel
	p.mu.Unlock()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.tailFile(tailCtx, s)
	}()
}

func (p *Provider) tailFile(ctx context.Context, s SessionFile) {
	f, err := os.Open(s.Path)
	if err != nil {
		slog.Warn("cannot open session file", "path", s.Path, "error", err)
		return
	}
	defer f.Close()

	// Seek to end — we only want new lines
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		slog.Warn("cannot seek session file", "path", s.Path, "error", err)
		return
	}

	reader := bufio.NewReader(f)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			// No more data — wait for fsnotify or poll
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
				continue
			}
		}

		if event := ParseJSONLLine(line, s.SessionID, s.Workspace); event != nil {
			p.bus.Publish(*event)
		}
	}
}

func (p *Provider) watchDirectories(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-p.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) {
				p.handleNewPath(ctx, event.Name)
			}
		case err, ok := <-p.watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("fsnotify error", "error", err)
		}
	}
}

func (p *Provider) handleNewPath(ctx context.Context, path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	if info.IsDir() {
		// New workspace directory — watch it and scan for sessions
		_ = p.watcher.Add(path)
		files, _ := os.ReadDir(path)
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".jsonl" {
				sf := SessionFile{
					Path:      filepath.Join(path, f.Name()),
					SessionID: f.Name()[:len(f.Name())-len(".jsonl")],
					Workspace: SlugToPath(filepath.Base(path)),
				}
				p.emitSessionStart(sf)
				p.startTailing(ctx, sf)
			}
		}
	} else if filepath.Ext(path) == ".jsonl" {
		dir := filepath.Dir(path)
		sf := SessionFile{
			Path:      path,
			SessionID: filepath.Base(path)[:len(filepath.Base(path))-len(".jsonl")],
			Workspace: SlugToPath(filepath.Base(dir)),
		}
		p.emitSessionStart(sf)
		p.startTailing(ctx, sf)
	}
}

func (p *Provider) pollProcesses(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			procs, err := DetectProcesses()
			if err != nil {
				slog.Warn("process detection failed", "error", err)
				continue
			}
			_ = procs // Process info available for cross-referencing
		}
	}
}
