package compat_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/compat"
	"opensteamtool-gui-mygo/internal/rt"
)

func newHost() *rt.Fake {
	return &rt.Fake{Files: map[string]bool{}, Contents: map[string][]byte{}, Answers: map[string]rt.Answer{}}
}

func writeDLL(t *testing.T, host *rt.Fake, dir string, name string, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	host.Files[path] = true
	host.Contents[path] = []byte(content)
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func rawURL(channel string, component string, sha string) string {
	return "https://raw.githubusercontent.com/OpenSteam001/steam-monitor/" + channel + "/" + component + "/" + sha + ".toml"
}

func mirrorURL(channel string, component string, sha string) string {
	return "https://fast.jsdelivr.net/gh/OpenSteam001/steam-monitor@" + channel + "/" + component + "/" + sha + ".toml"
}

func cacheFile(dir string, channel string, component string, sha string) string {
	return filepath.Join(dir, "opensteamtool", channel, component, sha+".toml")
}

func verifiedFile(dir string) string {
	return filepath.Join(dir, "cache", "verified.toml")
}

func steamWithCores(t *testing.T) (*rt.Fake, string, string, string) {
	t.Helper()
	host := newHost()
	steam := t.TempDir()
	client := writeDLL(t, host, steam, "steamclient64.dll", "client")
	ui := writeDLL(t, host, steam, "steamui.dll", "ui")
	return host, steam, client, ui
}

func answered(t *testing.T, host *rt.Fake, steam string, tool string) *compat.Round {
	t.Helper()
	round := compat.NewRound()
	job := round.Start(steam)
	if !round.Accept(job, compat.Probes(host, steam, tool, nil)) {
		t.Fatal("the round refused its own report")
	}
	return round
}

func TestAPublishedSignatureNeedsOneRequestAndIsNotCached(t *testing.T) {
	host, steam, client, ui := steamWithCores(t)
	for _, probe := range []struct{ channel, component, sha string }{
		{"pattern", "steamclient", client},
		{"pattern", "steamui", ui},
		{"ipc", "steamclient", client},
	} {
		host.Answers[rawURL(probe.channel, probe.component, probe.sha)] = rt.Answer{Body: []byte("sig")}
	}

	round := answered(t, host, steam, t.TempDir())
	if health := round.Snapshot().Health; health != app.HealthAdaptedNotCached {
		t.Fatalf("health = %v, want the adapted but uncached state", health)
	}
	if len(host.Fetched) != 3 {
		t.Fatalf("fetched %q, want one request per component", host.Fetched)
	}
	for _, url := range host.Fetched {
		if url != rawURL("pattern", "steamclient", client) && url != rawURL("pattern", "steamui", ui) && url != rawURL("ipc", "steamclient", client) {
			t.Fatalf("fetched %q, want the repository's raw endpoint first", host.Fetched)
		}
	}
}

func TestAnUnpublishedSignatureIsNotAdapted(t *testing.T) {
	host, steam, client, ui := steamWithCores(t)
	for _, probe := range []struct{ channel, component, sha string }{
		{"pattern", "steamclient", client},
		{"pattern", "steamui", ui},
		{"ipc", "steamclient", client},
	} {
		host.Answers[rawURL(probe.channel, probe.component, probe.sha)] = rt.Answer{Status: 404}
		host.Answers[mirrorURL(probe.channel, probe.component, probe.sha)] = rt.Answer{Status: 404}
	}

	round := answered(t, host, steam, t.TempDir())
	if health := round.Snapshot().Health; health != app.HealthNotAdapted {
		t.Fatalf("health = %v, want the not adapted state", health)
	}
	if len(host.Fetched) != 6 {
		t.Fatalf("fetched %q, want both mirrors tried for each component", host.Fetched)
	}
}

func TestA404ReadsAsUnadaptedAndNotOffline(t *testing.T) {
	host, steam, client, ui := steamWithCores(t)
	tool := t.TempDir()
	for _, probe := range []struct{ channel, component, sha string }{
		{"pattern", "steamclient", client},
		{"pattern", "steamui", ui},
		{"ipc", "steamclient", client},
	} {
		host.Answers[rawURL(probe.channel, probe.component, probe.sha)] = rt.Answer{Status: 404}
		path := cacheFile(steam, probe.channel, probe.component, probe.sha)
		host.Files[path] = true
		host.Contents[path] = []byte("sig")
	}

	round := answered(t, host, steam, tool)
	if health := round.Snapshot().Health; health != app.HealthNotAdapted {
		t.Fatalf("health = %v, want the unpublished conclusion, not offline", health)
	}
}

func TestOnlyAFullyUnreachableChainReportsOffline(t *testing.T) {
	host, steam, _, _ := steamWithCores(t)
	host.Answers = nil

	round := answered(t, host, steam, t.TempDir())
	if health := round.Snapshot().Health; health != app.HealthNetworkUnavailable {
		t.Fatalf("health = %v, want the network unavailable state", health)
	}
}

func TestAVerifiedFingerprintCarriesTheAdaptationOffline(t *testing.T) {
	host, steam, client, ui := steamWithCores(t)
	tool := t.TempDir()
	host.Files[verifiedFile(tool)] = true
	host.Contents[verifiedFile(tool)] = []byte("[verified]\nsteamclient_pattern = \"" + client + "\"\nsteamui_pattern = \"" + ui + "\"\nsteamclient_ipc = \"" + client + "\"\n")
	host.Answers = nil

	round := answered(t, host, steam, tool)
	if health := round.Snapshot().Health; health != app.HealthAdaptedNotCached {
		t.Fatalf("health = %v, want the recorded adaptation without a cache", health)
	}
	if fetched := host.Fetched; fetched == nil {
		t.Fatal("the probes did not reach the network")
	}
}

func TestAVerifiedFingerprintStopsMatchingWhenTheCoreChanges(t *testing.T) {
	host, steam, old, _ := steamWithCores(t)
	tool := t.TempDir()
	host.Files[verifiedFile(tool)] = true
	host.Contents[verifiedFile(tool)] = []byte("[verified]\nsteamclient_pattern = \"" + old + "\"\n")
	writeDLL(t, host, steam, "steamclient64.dll", "another client")
	host.Answers = nil

	round := answered(t, host, steam, tool)
	if health := round.Snapshot().Health; health != app.HealthNetworkUnavailable {
		t.Fatalf("health = %v, want no adaptation carried over from the old fingerprint", health)
	}
}

func TestAMissingCoreDllIsReportedPerComponent(t *testing.T) {
	host := newHost()
	steam := t.TempDir()
	ui := writeDLL(t, host, steam, "steamui.dll", "ui")

	round := answered(t, host, steam, t.TempDir())
	snapshot := round.Snapshot()
	if snapshot.Health != app.HealthMissing {
		t.Fatalf("health = %v, want the missing core state", snapshot.Health)
	}
	if len(snapshot.Components) != 3 {
		t.Fatalf("components = %+v, want one row per component", snapshot.Components)
	}
	want := []app.Component{
		{DLL: "steamclient64.dll", Kind: "Pattern", Health: app.HealthMissing},
		{DLL: "steamui.dll", Kind: "Pattern", Fingerprint: ui[:12], Health: app.HealthNetworkUnavailable},
		{DLL: "steamclient64.dll", Kind: "IPC", Health: app.HealthMissing},
	}
	if !slices.Equal(snapshot.Components, want) {
		t.Fatalf("components = %+v, want %+v", snapshot.Components, want)
	}
}

func TestALateReportNeverOverwritesANewerRound(t *testing.T) {
	_, steam, client, ui := steamWithCores(t)
	old := t.TempDir()
	round := compat.NewRound()
	first := round.Start(old)
	second := round.Start(steam)

	stale := compat.Report{Probes: []compat.Probe{
		{Target: compat.SteamclientPattern, Sha: client, Verdict: compat.Published},
		{Target: compat.SteamuiPattern, Sha: ui, Verdict: compat.Published},
		{Target: compat.SteamclientIpc, Sha: client, Verdict: compat.Published},
	}}
	if round.Accept(first, stale) {
		t.Fatal("a report from the old round was accepted")
	}
	if health := round.Snapshot().Health; health != app.HealthChecking {
		t.Fatalf("health = %v, want the newer round still checking", health)
	}
	if !round.Accept(second, stale) {
		t.Fatal("the current round refused its report")
	}
	if health := round.Snapshot().Health; health != app.HealthAdaptedNotCached {
		t.Fatalf("health = %v, want the current report", health)
	}
}

func TestAPrewarmCachesTheSignaturesAndRechecksWithoutTheNetwork(t *testing.T) {
	host, steam, client, ui := steamWithCores(t)
	tool := t.TempDir()
	for _, probe := range []struct{ channel, component, sha string }{
		{"pattern", "steamclient", client},
		{"pattern", "steamui", ui},
		{"ipc", "steamclient", client},
	} {
		host.Answers[rawURL(probe.channel, probe.component, probe.sha)] = rt.Answer{Body: []byte("sig")}
	}
	round := answered(t, host, steam, tool)

	job, probes, ok := round.BeginPrewarm(true)
	if !ok {
		t.Fatal("an adapted but uncached build offered no prewarm")
	}
	if _, _, ok := round.BeginPrewarm(true); ok {
		t.Fatal("the prewarm was started twice")
	}
	if snapshot := round.Snapshot(); !snapshot.Prewarming {
		t.Fatal("the prewarm is not visible while it runs")
	}
	done, err := compat.Prewarm(host, steam, tool, probes)
	if err != nil {
		t.Fatalf("prewarm = %v", err)
	}
	fetched := len(host.Fetched)
	if !round.Prewarmed(job, nil, done) {
		t.Fatal("the prewarm result was dropped")
	}

	snapshot := round.Snapshot()
	if snapshot.Health != app.HealthCompatibleCached {
		t.Fatalf("health = %v, want the cached compatibility within the same round", snapshot.Health)
	}
	if len(host.Fetched) != fetched {
		t.Fatalf("fetched %q after the prewarm, want no new request", host.Fetched[fetched:])
	}
	for _, probe := range []struct{ channel, component, sha string }{
		{"pattern", "steamclient", client},
		{"pattern", "steamui", ui},
		{"ipc", "steamclient", client},
	} {
		if !host.Files[cacheFile(steam, probe.channel, probe.component, probe.sha)] {
			t.Fatalf("cache %q was not written", cacheFile(steam, probe.channel, probe.component, probe.sha))
		}
	}
	body := string(host.Contents[verifiedFile(tool)])
	for _, want := range []string{"[verified]", `steamclient_pattern = "` + client, `steamui_pattern = "` + ui, `steamclient_ipc = "` + client} {
		if !strings.Contains(body, want) {
			t.Fatalf("verified cache = %q, want %q", body, want)
		}
	}
}

func TestAPrewarmLeavesASignatureWrittenForAnotherFingerprint(t *testing.T) {
	host, steam, client, ui := steamWithCores(t)
	tool := t.TempDir()
	for _, probe := range []struct{ channel, component, sha string }{
		{"pattern", "steamclient", client},
		{"pattern", "steamui", ui},
		{"ipc", "steamclient", client},
	} {
		host.Answers[rawURL(probe.channel, probe.component, probe.sha)] = rt.Answer{Body: []byte("sig")}
	}
	round := answered(t, host, steam, tool)
	_, probes, ok := round.BeginPrewarm(true)
	if !ok {
		t.Fatal("an adapted but uncached build offered no prewarm")
	}
	writeDLL(t, host, steam, "steamclient64.dll", "a newer client")

	if _, err := compat.Prewarm(host, steam, tool, probes); err != nil {
		t.Fatalf("prewarm = %v", err)
	}
	if host.Files[cacheFile(steam, "pattern", "steamclient", client)] {
		t.Fatal("a signature fetched for the old fingerprint was written")
	}
	if strings.Contains(string(host.Contents[verifiedFile(tool)]), client) {
		t.Fatalf("verified = %q, want the old fingerprint left out", host.Contents[verifiedFile(tool)])
	}
}

func TestAManualPrewarmFailureReportsItsCategory(t *testing.T) {
	host, steam, client, ui := steamWithCores(t)
	for _, probe := range []struct{ channel, component, sha string }{
		{"pattern", "steamclient", client},
		{"pattern", "steamui", ui},
		{"ipc", "steamclient", client},
	} {
		host.Answers[rawURL(probe.channel, probe.component, probe.sha)] = rt.Answer{Body: []byte("sig")}
	}
	round := answered(t, host, steam, t.TempDir())
	job, probes, ok := round.BeginPrewarm(true)
	if !ok {
		t.Fatal("an adapted but uncached build offered no prewarm")
	}
	host.Answers = nil

	if _, err := compat.Prewarm(host, steam, t.TempDir(), probes); err == nil {
		t.Fatal("a prewarm with no reachable mirror reported success")
	}
	if !round.Prewarmed(job, errors.New("unreachable"), nil) {
		t.Fatal("the prewarm result was dropped")
	}
	snapshot := round.Snapshot()
	if snapshot.Failure != app.NetworkFailed {
		t.Fatalf("failure = %v, want the network category", snapshot.Failure)
	}
	if snapshot.Prewarming || snapshot.Ready {
		t.Fatalf("snapshot = %+v, want the prewarm finished and not ready", snapshot)
	}
}

func TestAnAutomaticPrewarmFailureStaysSilent(t *testing.T) {
	host, steam, client, ui := steamWithCores(t)
	for _, probe := range []struct{ channel, component, sha string }{
		{"pattern", "steamclient", client},
		{"pattern", "steamui", ui},
		{"ipc", "steamclient", client},
	} {
		host.Answers[rawURL(probe.channel, probe.component, probe.sha)] = rt.Answer{Body: []byte("sig")}
	}
	round := answered(t, host, steam, t.TempDir())
	job, _, ok := round.BeginPrewarm(false)
	if !ok {
		t.Fatal("an adapted but uncached build offered no prewarm")
	}

	if !round.Prewarmed(job, errors.New("unreachable"), nil) {
		t.Fatal("the prewarm result was dropped")
	}
	snapshot := round.Snapshot()
	if snapshot.Failure != app.NoResult || snapshot.Ready {
		t.Fatalf("snapshot = %+v, want a silent failure with no state change", snapshot)
	}
	if snapshot.Health != app.HealthAdaptedNotCached {
		t.Fatalf("health = %v, want the prewarm entry still available", snapshot.Health)
	}
	if _, _, ok := round.BeginPrewarm(true); !ok {
		t.Fatal("the silent failure took the manual prewarm entry away")
	}
}

type clockHost struct {
	*rt.Fake
	at      time.Time
	onFetch func()
	timeout []time.Duration
}

func (h *clockHost) Fetch(url string, timeout time.Duration) (rt.Response, error) {
	h.timeout = append(h.timeout, timeout)
	if h.onFetch != nil {
		h.onFetch()
		h.onFetch = nil
	}
	return h.Fake.Fetch(url, timeout)
}

func TestTheRefreshSpendsOneBudgetAndThenReportsOffline(t *testing.T) {
	host, steam, _, _ := steamWithCores(t)
	host.Answers = nil
	clock := &clockHost{Fake: host, at: time.Unix(0, 0)}
	clock.onFetch = func() { clock.at = clock.at.Add(compat.Budget) }

	round := compat.NewRound()
	job := round.Start(steam)
	round.Accept(job, compat.Probes(clock, steam, t.TempDir(), func() time.Time { return clock.at }))

	if len(clock.timeout) != 1 {
		t.Fatalf("requests = %d, want one before the budget ran out", len(clock.timeout))
	}
	if clock.timeout[0] != compat.Budget {
		t.Fatalf("timeout = %v, want the whole budget", clock.timeout[0])
	}
	if health := round.Snapshot().Health; health != app.HealthNetworkUnavailable {
		t.Fatalf("health = %v, want the spent budget to conclude offline", health)
	}
}

func TestAnUnreadableCoreDllIsReportedAsALocalFailure(t *testing.T) {
	host, steam, _, _ := steamWithCores(t)
	client := filepath.Join(steam, "steamclient64.dll")
	delete(host.Contents, client)

	report := compat.Probes(host, steam, t.TempDir(), nil)
	if report.Err == nil {
		t.Fatal("an unreadable core DLL came back without an error")
	}
	if failure := compat.Category(report.Err); failure != app.LocalIOFailed {
		t.Errorf("category %v, want the local failure", failure)
	}
	if len(host.Fetched) != 2 {
		t.Errorf("fetched %q, want the steamui.dll probes to run anyway", host.Fetched)
	}
	round := compat.NewRound()
	round.Accept(round.Start(steam), report)
	if health := round.Snapshot().Health; health != app.HealthMissing {
		t.Errorf("health = %v, want the missing core state", health)
	}
}
