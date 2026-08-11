package testprogress

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const FilterEnv = "RENVO_TEST_FILTER"

type Logger interface {
	Helper()
	Logf(format string, args ...any)
}

type activeCase struct {
	name    string
	started time.Time
}

type completedCase struct {
	name     string
	duration time.Duration
}

type groupProgress struct {
	name     string
	cases    int
	duration time.Duration
}

type Progress struct {
	logger  Logger
	label   string
	total   int
	started time.Time

	mu        sync.Mutex
	active    map[string]time.Time
	completed int
	slowest   []completedCase
	groups    map[string]groupProgress
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func New(logger Logger, label string, total int) *Progress {
	logger.Helper()
	p := &Progress{
		logger: logger, label: label, total: total, started: time.Now(),
		active: make(map[string]time.Time), groups: make(map[string]groupProgress),
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	logger.Logf("%s: starting %d cases", label, total)
	go p.reportLoop()
	return p
}

func (p *Progress) Begin(name string) func() {
	started := time.Now()
	p.mu.Lock()
	p.active[name] = started
	p.mu.Unlock()
	return func() {
		duration := time.Since(started)
		p.mu.Lock()
		delete(p.active, name)
		p.completed++
		p.slowest = append(p.slowest, completedCase{name: name, duration: duration})
		sort.Slice(p.slowest, func(i, j int) bool { return p.slowest[i].duration > p.slowest[j].duration })
		if len(p.slowest) > 5 {
			p.slowest = p.slowest[:5]
		}
		groupName := caseGroup(name)
		group := p.groups[groupName]
		group.name = groupName
		group.cases++
		group.duration += duration
		p.groups[groupName] = group
		p.mu.Unlock()
	}
}

func (p *Progress) Close() {
	p.closeOnce.Do(func() {
		close(p.stop)
		<-p.done
		p.mu.Lock()
		completed := p.completed
		slowest := append([]completedCase(nil), p.slowest...)
		groups := make([]groupProgress, 0, len(p.groups))
		for _, group := range p.groups {
			groups = append(groups, group)
		}
		p.mu.Unlock()
		sort.Slice(groups, func(i, j int) bool { return groups[i].duration > groups[j].duration })
		if len(groups) > 5 {
			groups = groups[:5]
		}
		p.logger.Logf("%s: completed %d/%d cases in %s; slowest cases: %s; slowest groups: %s",
			p.label, completed, p.total, elapsed(time.Since(p.started)), formatCompleted(slowest), formatGroups(groups))
	})
}

func (p *Progress) reportLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer func() {
		ticker.Stop()
		close(p.done)
	}()
	for {
		select {
		case <-ticker.C:
			p.report()
		case <-p.stop:
			return
		}
	}
}

func (p *Progress) report() {
	now := time.Now()
	p.mu.Lock()
	completed := p.completed
	active := make([]activeCase, 0, len(p.active))
	for name, started := range p.active {
		active = append(active, activeCase{name: name, started: started})
	}
	p.mu.Unlock()
	sort.Slice(active, func(i, j int) bool { return active[i].started.Before(active[j].started) })
	if len(active) > 3 {
		active = active[:3]
	}
	var names []string
	for _, item := range active {
		names = append(names, fmt.Sprintf("%s (%s)", item.name, elapsed(now.Sub(item.started))))
	}
	p.logger.Logf("%s: %d/%d complete after %s; oldest active: %s",
		p.label, completed, p.total, elapsed(now.Sub(p.started)), strings.Join(names, ", "))
}

func Filter(names []string, expression string) ([]string, error) {
	if expression == "" {
		return append([]string(nil), names...), nil
	}
	pattern, err := regexp.Compile(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid %s regular expression %q: %w", FilterEnv, expression, err)
	}
	var selected []string
	for _, name := range names {
		if pattern.MatchString(filepathSlash(name)) {
			selected = append(selected, name)
		}
	}
	return selected, nil
}

func filepathSlash(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func formatCompleted(cases []completedCase) string {
	if len(cases) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(cases))
	for _, item := range cases {
		parts = append(parts, fmt.Sprintf("%s=%s", item.name, elapsed(item.duration)))
	}
	return strings.Join(parts, ", ")
}

func formatGroups(groups []groupProgress) string {
	if len(groups) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(groups))
	for _, group := range groups {
		parts = append(parts, fmt.Sprintf("%s=%s/%d", group.name, elapsed(group.duration), group.cases))
	}
	return strings.Join(parts, ", ")
}

func caseGroup(name string) string {
	normalized := filepathSlash(name)
	parts := strings.Split(normalized, "/")
	if len(parts) > 1 && parts[0] != "tests" {
		return parts[0]
	}
	base := parts[len(parts)-1]
	base = strings.TrimSuffix(base, ".go")
	if split := strings.IndexByte(base, '_'); split > 0 {
		return base[:split]
	}
	return base
}

func elapsed(duration time.Duration) string {
	return duration.Round(10 * time.Millisecond).String()
}
