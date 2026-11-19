package main

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
)

func compatHost(t *testing.T) (*rt.Fake, string, string, string) {
	t.Helper()
	root := t.TempDir()
	host := &rt.Fake{
		Dirs:     map[string]bool{root: true},
		Files:    map[string]bool{},
		Contents: map[string][]byte{},
		Answers:  map[string]rt.Answer{},
	}
	host.Files[filepath.Join(root, "steam.exe")] = true
	sums := map[string]string{}
	for _, name := range []string{"steamclient64.dll", "steamui.dll"} {
		path := filepath.Join(root, name)
		host.Files[path] = true
		host.Contents[path] = []byte(name)
		sum := sha256.Sum256([]byte(name))
		sums[name] = hex.EncodeToString(sum[:])
	}
	return host, root, sums["steamclient64.dll"], sums["steamui.dll"]
}

func publishSignatures(host *rt.Fake, client string, ui string) {
	for _, path := range []string{
		"pattern/steamclient/" + client + ".toml",
		"pattern/steamui/" + ui + ".toml",
		"ipc/steamclient/" + client + ".toml",
	} {
		host.Answers["https://raw.githubusercontent.com/OpenSteam001/steam-monitor/"+path] = rt.Answer{Body: []byte("sig")}
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestAnAdaptedBuildCachesItselfWithoutABlockingCommand(t *testing.T) {
	host, root, client, ui := compatHost(t)
	publishSignatures(host, client, ui)
	s := newTestSession(t, host, root)

	s.startCompat()
	waitFor(t, "the cached state", func() bool {
		return s.factsWithoutGate().Health == app.HealthCompatibleCached
	})

	snapshot := s.snapshot()
	if snapshot.Busy != app.Idle {
		t.Errorf("the cache ran inside the gate as %v", snapshot.Busy)
	}
	if app.Derive(snapshot).Risk {
		t.Error("a cached build raises the risk line")
	}
	if !host.Files[filepath.Join(root, "opensteamtool", "pattern", "steamclient", client+".toml")] {
		t.Error("the signature is not in the Steam directory's cache")
	}
	if !host.Files[filepath.Join(s.dir, "cache", "verified.toml")] {
		t.Error("the verification cache is missing from the tool directory")
	}
}

func TestANewRoundReadsAsChecking(t *testing.T) {
	host, root, _, _ := compatHost(t)
	s := newTestSession(t, host, root)
	s.facts.Health = app.HealthCompatibleCached

	s.startCompat()
	if health := s.factsWithoutGate().Health; health != app.HealthChecking {
		t.Errorf("health = %v, want the new round reported as checking", health)
	}
}

func TestAnUnreachableNetworkOnlyReportsItself(t *testing.T) {
	host, root, _, _ := compatHost(t)
	s := newTestSession(t, host, root)

	s.startCompat()
	waitFor(t, "the offline conclusion", func() bool {
		return s.factsWithoutGate().Health == app.HealthNetworkUnavailable
	})
	if s.factsWithoutGate().PrecacheFailure != app.NoResult {
		t.Errorf("the automatic attempt reported %v", s.factsWithoutGate().PrecacheFailure)
	}
	if app.Derive(s.snapshot()).Risk {
		t.Error("an unreachable network raises the risk line")
	}
}

func TestAManualPrewarmFailureNamesItsCategory(t *testing.T) {
	host, root, _, _ := compatHost(t)
	s := newTestSession(t, host, root)

	s.startCompat()
	waitFor(t, "a reported conclusion", func() bool {
		return s.factsWithoutGate().Health == app.HealthNetworkUnavailable
	})
	s.prewarm(true)
	waitFor(t, "the reported failure", func() bool {
		return s.factsWithoutGate().PrecacheFailure == app.NetworkFailed
	})
	if snapshot := s.snapshot(); snapshot.Prewarming || snapshot.PrecacheReady {
		t.Errorf("snapshot = %+v, want the attempt finished and not ready", snapshot)
	}
	if result := s.factsWithoutGate().Result; result != app.NetworkFailed {
		t.Errorf("status bar = %v, want the failure the user asked for", result)
	}
}

func TestOpeningTheCompatibilityDetailsLooksNothingUp(t *testing.T) {
	host, root, _, _ := compatHost(t)
	s := newTestSession(t, host, root)

	s.startCompat()
	waitFor(t, "the offline conclusion", func() bool {
		return s.factsWithoutGate().Health == app.HealthNetworkUnavailable
	})
	fetched := len(host.Fetched)
	s.uiState.CompatDetails = true
	tt := ui.NewTester(s.view, 480, 720)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if !tt.HasText("steamclient64.dll（Pattern）") {
		t.Fatalf("texts %q without the open details", tt.Texts())
	}
	if len(host.Fetched) != fetched {
		t.Errorf("opening the details fetched %q", host.Fetched[fetched:])
	}
}
