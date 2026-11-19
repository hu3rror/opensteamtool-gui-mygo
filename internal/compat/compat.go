package compat

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
)

const Budget = 2500 * time.Millisecond

type Host interface {
	IsFile(path string) bool
	ReadFile(path string) ([]byte, error)
	MkdirAll(path string) error
	WriteFile(path string, data []byte) error
	Rename(from string, to string) error
	Fetch(url string, timeout time.Duration) (rt.Response, error)
}

type Target int

const (
	SteamclientPattern Target = iota
	SteamuiPattern
	SteamclientIpc
)

var targets = []Target{SteamclientPattern, SteamuiPattern, SteamclientIpc}

func (t Target) dll() string {
	if t == SteamuiPattern {
		return "steamui.dll"
	}
	return "steamclient64.dll"
}

func (t Target) kind() string {
	if t == SteamclientIpc {
		return "IPC"
	}
	return "Pattern"
}

func (t Target) channel() string {
	if t == SteamclientIpc {
		return "ipc"
	}
	return "pattern"
}

func (t Target) component() string {
	if t == SteamuiPattern {
		return "steamui"
	}
	return "steamclient"
}

func (t Target) key() string {
	switch t {
	case SteamuiPattern:
		return "steamui_pattern"
	case SteamclientIpc:
		return "steamclient_ipc"
	}
	return "steamclient_pattern"
}

type Verdict int

const (
	Unasked Verdict = iota
	Published
	Absent
	Unreachable
)

type Probe struct {
	Target   Target
	Sha      string
	Verdict  Verdict
	Cached   bool
	Verified bool
}

type Report struct {
	Probes []Probe
	Err    error
}

type Failure int

const (
	NetworkFailure Failure = iota
	LocalFailure
)

type Error struct {
	Failure Failure
	Err     error
}

func (e *Error) Error() string { return e.Err.Error() }

func (e *Error) Unwrap() error { return e.Err }

func Category(err error) app.Result {
	var failure *Error
	if errors.As(err, &failure) && failure.Failure == LocalFailure {
		return app.LocalIOFailed
	}
	return app.NetworkFailed
}

func Probes(host Host, steamDir string, toolDir string, now func() time.Time) Report {
	if now == nil {
		now = time.Now
	}
	report := Report{Probes: make([]Probe, 0, len(targets))}
	fingerprints := map[string]string{}
	for _, name := range []string{"steamclient64.dll", "steamui.dll"} {
		sha, err := fingerprint(host, filepath.Join(steamDir, name))
		if err != nil && report.Err == nil {
			report.Err = err
		}
		fingerprints[name] = sha
	}
	verified := readVerified(host, toolDir)
	deadline := now().Add(Budget)
	for _, target := range targets {
		probe := Probe{Target: target, Sha: fingerprints[target.dll()]}
		if probe.Sha != "" {
			probe.Cached = host.IsFile(cachePath(steamDir, target, probe.Sha))
			probe.Verified = verified[target.key()] == probe.Sha
			probe.Verdict = ask(host, target, probe.Sha, deadline, now)
		}
		report.Probes = append(report.Probes, probe)
	}
	return report
}

func Prewarm(host Host, steamDir string, toolDir string, probes []Probe) ([]Probe, error) {
	verified := readVerified(host, toolDir)
	done := make([]Probe, len(probes))
	copy(done, probes)
	dirty := false
	for i, probe := range done {
		if probe.Sha == "" || probe.Cached || probe.Verdict == Absent {
			continue
		}
		sha, err := fingerprint(host, filepath.Join(steamDir, probe.Target.dll()))
		if err != nil || sha != probe.Sha {
			continue
		}
		body, err := fetchBody(host, probe.Target, probe.Sha)
		if err != nil {
			return done, err
		}
		if err := writeCache(host, steamDir, probe.Target, probe.Sha, body); err != nil {
			return done, err
		}
		verified[probe.Target.key()] = probe.Sha
		done[i].Cached = true
		done[i].Verified = true
		dirty = true
	}
	if !dirty {
		return done, nil
	}
	return done, writeVerified(host, toolDir, verified)
}

func fingerprint(host Host, path string) (string, error) {
	data, err := host.ReadFile(path)
	if err != nil {
		if !host.IsFile(path) {
			return "", nil
		}
		return "", &Error{Failure: LocalFailure, Err: err}
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func ask(host Host, target Target, sha string, deadline time.Time, now func() time.Time) Verdict {
	absent := false
	for _, url := range urls(target, sha) {
		left := deadline.Sub(now())
		if left <= 0 {
			break
		}
		response, err := host.Fetch(url, left)
		if err != nil {
			continue
		}
		switch response.Status {
		case 200:
			return Published
		case 404:
			absent = true
		}
	}
	if absent {
		return Absent
	}
	return Unreachable
}

func fetchBody(host Host, target Target, sha string) ([]byte, error) {
	for _, url := range urls(target, sha) {
		response, err := host.Fetch(url, Budget)
		if err != nil {
			continue
		}
		if response.Status == 200 {
			return response.Body, nil
		}
	}
	return nil, &Error{Failure: NetworkFailure, Err: errors.New("no mirror published the signature")}
}

func urls(target Target, sha string) []string {
	path := target.channel() + "/" + target.component() + "/" + sha + ".toml"
	return []string{
		"https://raw.githubusercontent.com/OpenSteam001/steam-monitor/" + path,
		"https://fast.jsdelivr.net/gh/OpenSteam001/steam-monitor@" + path,
	}
}

func cachePath(steamDir string, target Target, sha string) string {
	return filepath.Join(steamDir, "opensteamtool", target.channel(), target.component(), sha+".toml")
}

func verifiedPath(toolDir string) string {
	return filepath.Join(toolDir, "cache", "verified.toml")
}

func readVerified(host Host, toolDir string) map[string]string {
	entries := map[string]string{}
	data, err := host.ReadFile(verifiedPath(toolDir))
	if err != nil {
		return entries
	}
	for _, line := range strings.Split(string(data), "\n") {
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		entries[strings.TrimSpace(name)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return entries
}

func writeVerified(host Host, toolDir string, entries map[string]string) error {
	text := "[verified]\n"
	for _, target := range targets {
		if sha := entries[target.key()]; sha != "" {
			text += target.key() + " = \"" + sha + "\"\n"
		}
	}
	path := verifiedPath(toolDir)
	if err := host.MkdirAll(filepath.Dir(path)); err != nil {
		return &Error{Failure: LocalFailure, Err: err}
	}
	if err := host.WriteFile(path, []byte(text)); err != nil {
		return &Error{Failure: LocalFailure, Err: err}
	}
	return nil
}

func writeCache(host Host, steamDir string, target Target, sha string, body []byte) error {
	path := cachePath(steamDir, target, sha)
	if err := host.MkdirAll(filepath.Dir(path)); err != nil {
		return &Error{Failure: LocalFailure, Err: err}
	}
	part := path + ".part"
	if err := host.WriteFile(part, body); err != nil {
		return &Error{Failure: LocalFailure, Err: err}
	}
	if err := host.Rename(part, path); err != nil {
		return &Error{Failure: LocalFailure, Err: err}
	}
	return nil
}

type Job struct {
	Epoch int
	Path  string
}

type Snapshot struct {
	Health     app.Health
	Components []app.Component
	Prewarming bool
	Ready      bool
	Failure    app.Result
}

type Round struct {
	epoch    int
	path     string
	report   Report
	answered bool

	prewarming bool
	manual     bool
	ready      bool
	failure    app.Result
}

func NewRound() *Round {
	return &Round{}
}

func (r *Round) Start(path string) Job {
	r.path = path
	r.epoch++
	r.report = Report{}
	r.answered = false
	r.prewarming = false
	r.ready = false
	r.failure = app.NoResult
	return Job{Epoch: r.epoch, Path: path}
}

func (r *Round) Accept(job Job, report Report) bool {
	if job.Epoch != r.epoch {
		return false
	}
	r.path = job.Path
	r.report = report
	r.answered = true
	return true
}

func (r *Round) Snapshot() Snapshot {
	snapshot := Snapshot{Health: app.HealthChecking, Prewarming: r.prewarming, Ready: r.ready, Failure: r.failure}
	if !r.answered {
		return snapshot
	}
	snapshot.Health = healthOf(r.report)
	snapshot.Components = componentsOf(r.report)
	return snapshot
}

func (r *Round) BeginPrewarm(manual bool) (Job, []Probe, bool) {
	if !r.answered || r.prewarming {
		return Job{}, nil, false
	}
	probes := []Probe{}
	for _, probe := range r.report.Probes {
		if probe.Sha == "" || probe.Cached || probe.Verdict == Absent {
			continue
		}
		probes = append(probes, probe)
	}
	if len(probes) == 0 {
		return Job{}, nil, false
	}
	r.prewarming = true
	r.manual = manual
	r.ready = false
	r.failure = app.NoResult
	return Job{Epoch: r.epoch, Path: r.path}, probes, true
}

func (r *Round) Prewarmed(job Job, err error, done []Probe) bool {
	if job.Epoch != r.epoch || !r.prewarming {
		return false
	}
	r.prewarming = false
	r.ready = err == nil
	r.failure = app.NoResult
	if err != nil && r.manual {
		r.failure = Category(err)
	}
	for _, probe := range done {
		for i := range r.report.Probes {
			if r.report.Probes[i].Target == probe.Target {
				r.report.Probes[i].Cached = probe.Cached
				r.report.Probes[i].Verified = probe.Verified
			}
		}
	}
	return true
}

var healthRanks = map[app.Health]int{
	app.HealthCompatibleCached:   0,
	app.HealthAdaptedNotCached:   1,
	app.HealthOfflineReady:       2,
	app.HealthNetworkUnavailable: 3,
	app.HealthNotAdapted:         4,
	app.HealthMissing:            5,
}

func healthOf(report Report) app.Health {
	worst := app.HealthCompatibleCached
	for _, probe := range report.Probes {
		if health := probeHealth(probe); healthRanks[health] > healthRanks[worst] {
			worst = health
		}
	}
	return worst
}

func probeHealth(probe Probe) app.Health {
	upstream := probe.Verdict == Published || probe.Verified
	switch {
	case probe.Sha == "":
		return app.HealthMissing
	case probe.Verdict == Absent:
		return app.HealthNotAdapted
	case probe.Cached && upstream:
		return app.HealthCompatibleCached
	case !probe.Cached && upstream:
		return app.HealthAdaptedNotCached
	case probe.Cached:
		return app.HealthOfflineReady
	}
	return app.HealthNetworkUnavailable
}

func componentsOf(report Report) []app.Component {
	components := make([]app.Component, 0, len(report.Probes))
	for _, probe := range report.Probes {
		components = append(components, app.Component{
			DLL:         probe.Target.dll(),
			Kind:        probe.Target.kind(),
			Fingerprint: head(probe.Sha),
			Health:      probeHealth(probe),
		})
	}
	return components
}

func head(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
