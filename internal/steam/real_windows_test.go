//go:build acceptreal

package steam_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/steam"
)

func writeSteam(t *testing.T, root, source string) {
	t.Helper()
	src := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(src, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(root, steam.Exe), src)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, out)
	}
}

func TestRealStopForceKillsAfterTheGracefulBudget(t *testing.T) {
	root := t.TempDir()
	writeSteam(t, root, "package main\n\nimport (\n\t\"os\"\n\t\"time\"\n)\n\nfunc main() {\n\tif len(os.Args) > 1 {\n\t\treturn\n\t}\n\ttime.Sleep(time.Hour)\n}\n")

	client := steam.New(rt.OS{}, time.Sleep)
	if err := client.Start(root); err != nil {
		t.Fatalf("start: %v", err)
	}
	begin := time.Now()
	err := client.Stop(root)
	elapsed := time.Since(begin)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if elapsed < steam.GracefulBudget {
		t.Fatalf("stopped in %v, before the %v graceful budget", elapsed, steam.GracefulBudget)
	}
	if elapsed > steam.GracefulBudget+steam.ForceBudget+2*time.Second {
		t.Fatalf("stopped in %v, after both budgets", elapsed)
	}
	running, err := steam.Running(rt.OS{}, root)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("group is not empty after the force budget")
	}
}

func TestRealStartFailsWhenTheProgramExitsAtOnce(t *testing.T) {
	root := t.TempDir()
	writeSteam(t, root, "package main\n\nfunc main() {}\n")

	client := steam.New(rt.OS{}, time.Sleep)
	begin := time.Now()
	err := client.Start(root)
	elapsed := time.Since(begin)
	if err != steam.ErrNotStarted {
		t.Fatalf("start returned %v, want ErrNotStarted", err)
	}
	if elapsed < 2*steam.LaunchCheck+steam.LaunchRetry {
		t.Fatalf("gave up in %v, before the retry", elapsed)
	}
	if elapsed > 2*steam.LaunchCheck+steam.LaunchRetry+2*time.Second {
		t.Fatalf("gave up in %v, after the retry budget", elapsed)
	}
}
