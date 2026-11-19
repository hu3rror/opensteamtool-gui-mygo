package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
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
		ModTimes: map[string]time.Time{},
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
	host, root, s := settledSession(t)
	release := holdFetch(host)
	defer release()
	if err := host.WriteFile(filepath.Join(root, "steamui.dll"), []byte("a patched steamui.dll")); err != nil {
		t.Fatal(err)
	}

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

func writeClientBuild(t *testing.T, steamDir string, build string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(steamDir, "package"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "{\n\t\"version\"\t\"" + build + "\"\n}\n"
	if err := os.WriteFile(filepath.Join(steamDir, "package", "steam_client_win32.manifest"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func settledSession(t *testing.T) (*rt.Fake, string, *session) {
	t.Helper()
	return settledSessionWithBuild(t, "")
}

func settledSessionWithBuild(t *testing.T, build string) (*rt.Fake, string, *session) {
	t.Helper()
	host, root, client, ui := compatHost(t)
	publishSignatures(host, client, ui)
	if build != "" {
		writeClientBuild(t, root, build)
	}
	s := newTestSession(t, host, root)
	s.startCompat()
	waitFor(t, "the cached state", func() bool {
		return s.factsWithoutGate().Health == app.HealthCompatibleCached
	})
	return host, root, s
}

func holdFetch(host *rt.Fake) func() {
	gate := make(chan struct{})
	released := false
	host.OnFetch = func(string) { <-gate }
	return func() {
		if !released {
			released = true
			close(gate)
		}
	}
}

func TestTheCompatStampFollowsTheCoreLibraries(t *testing.T) {
	host, root, _, _ := compatHost(t)
	stamp := compatStamp(host, root)

	if again := compatStamp(host, root); again != stamp {
		t.Error("an idle Steam directory changed the stamp")
	}
	host.ModTimes[filepath.Join(root, "steamui.dll")] = time.Unix(0, 1)
	if rewritten := compatStamp(host, root); rewritten == stamp {
		t.Error("a rewritten library of the same size kept the stamp")
	}
}

func TestATickThatChangesNothingStartsNoRound(t *testing.T) {
	host, _, s := settledSession(t)
	fetched := len(host.Fetched)
	reads := len(host.Reads)

	s.startCompat()
	if health := s.factsWithoutGate().Health; health != app.HealthCompatibleCached {
		t.Errorf("health = %v, want the settled conclusion kept", health)
	}
	if len(host.Fetched) != fetched || len(host.Reads) != reads {
		t.Errorf("a tick without a change fetched %q and read %q", host.Fetched[fetched:], host.Reads[reads:])
	}
}

func TestACoreLibraryChangeStartsANewRoundWithoutTheManifest(t *testing.T) {
	host, root, s := settledSession(t)
	if _, err := os.Stat(filepath.Join(root, "package", "steam_client_win32.manifest")); err == nil {
		t.Error("the test left a client manifest behind")
	}
	fetched := len(host.Fetched)
	if err := host.WriteFile(filepath.Join(root, "steamclient64.dll"), []byte("a patched steamclient64.dll")); err != nil {
		t.Fatal(err)
	}

	s.startCompat()
	waitFor(t, "the new round's lookups", func() bool {
		return len(host.Fetched) > fetched
	})
}

func TestAClientBuildChangeStartsANewRound(t *testing.T) {
	host, root, s := settledSessionWithBuild(t, "1769731672")
	fetched := len(host.Fetched)
	writeClientBuild(t, root, "1769731673")

	s.startCompat()
	waitFor(t, "the new round's lookups", func() bool {
		return len(host.Fetched) > fetched
	})
}
