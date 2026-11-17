package steam_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/steam"
)

const root = `C:\Steam`

type trace struct {
	total  time.Duration
	events []string
}

func (t *trace) sleep(d time.Duration) {
	t.total += d
	t.events = append(t.events, "sleep")
}

func (t *trace) first(event string) int {
	return slices.Index(t.events, event)
}

func clientWith(fake *rt.Fake) (*steam.Client, *trace) {
	t := &trace{}
	return steam.New(fake, t.sleep), t
}

func running(t *testing.T, fake *rt.Fake, path string) bool {
	t.Helper()
	on, err := steam.Running(fake, path)
	if err != nil {
		t.Fatalf("cannot read the process list: %v", err)
	}
	return on
}

func steamProcess(pid int) rt.Process {
	return rt.Process{PID: pid, Exe: root + `\steam.exe`}
}

func TestStopSucceedsAtOnceWithoutASteamProcess(t *testing.T) {
	fake := &rt.Fake{Procs: []rt.Process{{PID: 7, Exe: `C:\Games\game.exe`}}}
	client, t2 := clientWith(fake)

	if err := client.Stop(root); err != nil {
		t.Fatalf("Stop reported %v without a Steam process", err)
	}
	if len(fake.Spawned) != 0 || len(fake.Killed) != 0 {
		t.Fatalf("Stop touched processes: spawned %v, terminated %v", fake.Spawned, fake.Killed)
	}
	if t2.total != 0 {
		t.Errorf("Stop waited %v without a Steam process", t2.total)
	}
}

func TestStopAsksTheMainProgramToExitBeforeForcing(t *testing.T) {
	fake := &rt.Fake{Files: map[string]bool{root + `\steam.exe`: true}, Procs: []rt.Process{steamProcess(100)}}
	client, t2 := clientWith(fake)
	fake.OnSpawn = func(call rt.Launch) {
		t2.events = append(t2.events, "spawn "+strings.Join(call.Args, " "))
		fake.Procs = nil
	}

	if err := client.Stop(root); err != nil {
		t.Fatalf("Stop reported %v while the program exited on its own", err)
	}
	if len(fake.Spawned) != 1 {
		t.Fatalf("Stop spawned the main program %d times, want once", len(fake.Spawned))
	}
	call := fake.Spawned[0]
	if call.Exe != root+`\steam.exe` || call.Dir != root || !slices.Equal(call.Args, []string{"-shutdown"}) {
		t.Errorf("Stop spawned %q in %q with %v, want %q in %q with -shutdown", call.Exe, call.Dir, call.Args, root+`\steam.exe`, root)
	}
	if len(fake.Killed) != 0 {
		t.Errorf("Stop forced processes that exited gracefully: %v", fake.Killed)
	}
}

func TestStopForcesOnlyAfterTheGracefulBudget(t *testing.T) {
	fake := &rt.Fake{Files: map[string]bool{root + `\steam.exe`: true}, Procs: []rt.Process{steamProcess(100)}}
	client, t2 := clientWith(fake)
	fake.OnTerminate = func(pid int) {
		t2.events = append(t2.events, "terminate")
		fake.Procs = slices.DeleteFunc(fake.Procs, func(p rt.Process) bool { return p.PID == pid })
	}

	if err := client.Stop(root); err != nil {
		t.Fatalf("Stop reported %v after the forced exit", err)
	}
	if !slices.Equal(fake.Killed, []int{100}) {
		t.Fatalf("Stop terminated %v, want the stubborn main program", fake.Killed)
	}
	if t2.total < steam.GracefulBudget {
		t.Errorf("Stop forced after %v, want at least %v of graceful time", t2.total, steam.GracefulBudget)
	}
	if t2.total > steam.GracefulBudget+steam.ForceBudget {
		t.Errorf("Stop waited %v, more than both budgets", t2.total)
	}
	if waits := t2.first("terminate"); waits < int(steam.GracefulBudget/steam.Step) {
		t.Errorf("Stop forced after %d polls, want the %v budget first", waits, steam.GracefulBudget)
	}
}

func TestStopForcesAtOnceWhenTheMainProgramCannotBeAsked(t *testing.T) {
	fake := &rt.Fake{Procs: []rt.Process{
		steamProcess(100),
		{PID: 200, Exe: root + `\bin\cef\cef.win64\steamwebhelper.exe`},
	}}
	client, t2 := clientWith(fake)

	if err := client.Stop(root); err != nil {
		t.Fatalf("Stop reported %v although the forced group emptied", err)
	}
	if len(fake.Spawned) != 0 {
		t.Errorf("Stop spawned %v without a steam.exe on disk", fake.Spawned)
	}
	if !slices.Equal(fake.Killed, []int{100, 200}) {
		t.Errorf("Stop terminated %v, want the whole group", fake.Killed)
	}
	if t2.total != 0 {
		t.Errorf("Stop waited %v although it could not ask for an exit", t2.total)
	}
}

func TestStopNeverForcesTheHelperItStarted(t *testing.T) {
	fake := &rt.Fake{Files: map[string]bool{root + `\steam.exe`: true}, Procs: []rt.Process{steamProcess(100)}}
	client, _ := clientWith(fake)
	fake.OnSpawn = func(call rt.Launch) {
		fake.Procs = slices.DeleteFunc(fake.Procs, func(p rt.Process) bool { return p.PID == 100 })
		fake.Procs = append(fake.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}

	if err := client.Stop(root); err != nil {
		t.Fatalf("Stop reported %v although the main program exited", err)
	}
	if len(fake.Killed) != 0 {
		t.Errorf("Stop forced %v, only its own shutdown helper was left", fake.Killed)
	}
}

func TestStopFailsWhenTheGroupNeverEmpties(t *testing.T) {
	fake := &rt.Fake{Files: map[string]bool{root + `\steam.exe`: true}, Procs: []rt.Process{steamProcess(100)}}
	client, t2 := clientWith(fake)
	fake.OnTerminate = func(int) {}

	err := client.Stop(root)
	if !errors.Is(err, steam.ErrNotStopped) {
		t.Fatalf("Stop reported %v, want ErrNotStopped", err)
	}
	if t2.total < steam.GracefulBudget+steam.ForceBudget {
		t.Errorf("Stop gave up after %v, want both budgets spent", t2.total)
	}
	if len(fake.Killed) == 0 {
		t.Error("Stop never forced the group")
	}
}

func TestStopCountsOnlySteamProgramsUnderTheDirectory(t *testing.T) {
	fake := &rt.Fake{
		Files: map[string]bool{root + `\steam.exe`: true},
		Procs: []rt.Process{
			steamProcess(100),
			{PID: 200, Exe: root + `\bin\cef\cef.win64\steamwebhelper.exe`},
			{PID: 300, Exe: root + `\steamservice.exe`},
			{PID: 400, Exe: `C:\Program Files (x86)\Common Files\Steam\steamservice.exe`},
			{PID: 500, Exe: `C:\SteamLibrary\steam.exe`},
			{PID: 600, Exe: `C:\Games\game.exe`},
		},
	}
	client, _ := clientWith(fake)
	fake.OnSpawn = func(rt.Launch) {}

	if !running(t, fake, root) {
		t.Fatal("Running reports no Steam while steam.exe runs")
	}
	if err := client.Stop(root); err != nil {
		t.Fatalf("Stop reported %v although the forced group emptied", err)
	}
	if !slices.Equal(fake.Killed, []int{100, 200}) {
		t.Errorf("Stop terminated %v, want only the Steam program and its helper", fake.Killed)
	}
	if len(fake.Spawned) != 1 {
		t.Errorf("Stop spawned %d processes, want only the main program", len(fake.Spawned))
	}
}

func TestTheGroupFollowsTheDirectoryWhateverItsCaseAndEnding(t *testing.T) {
	fake := &rt.Fake{Procs: []rt.Process{{PID: 100, Exe: `C:\Steam\steam.exe`}}}

	for _, path := range []string{`c:\steam`, `C:\Steam\`, `C:/Steam`} {
		if !running(t, fake, path) {
			t.Errorf("Running does not see steam.exe under %q", path)
		}
	}
	for _, path := range []string{"", `C:\SteamLibrary`, `C:\Steam\bin`} {
		if running(t, fake, path) {
			t.Errorf("Running sees steam.exe under %q", path)
		}
	}
}

func TestStartSucceedsWhenTheProgramIsAliveAfterTwoSeconds(t *testing.T) {
	fake := &rt.Fake{Files: map[string]bool{root + `\steam.exe`: true}}
	client, t2 := clientWith(fake)
	fake.OnSpawn = func(call rt.Launch) {
		fake.Procs = append(fake.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}

	if err := client.Start(root); err != nil {
		t.Fatalf("Start reported %v for a live program", err)
	}
	if len(fake.Spawned) != 1 {
		t.Fatalf("Start spawned %d times, want once", len(fake.Spawned))
	}
	call := fake.Spawned[0]
	if call.Exe != root+`\steam.exe` || call.Dir != root || len(call.Args) != 0 {
		t.Errorf("Start spawned %q in %q with %v, want the bare main program", call.Exe, call.Dir, call.Args)
	}
	if t2.total != steam.LaunchCheck {
		t.Errorf("Start waited %v, want %v", t2.total, steam.LaunchCheck)
	}
}

func TestStartRetriesOnceWhenTheProgramLeavesAtOnce(t *testing.T) {
	fake := &rt.Fake{Files: map[string]bool{root + `\steam.exe`: true}}
	client, t2 := clientWith(fake)
	fake.OnSpawn = func(rt.Launch) {}

	if err := client.Start(root); !errors.Is(err, steam.ErrNotStarted) {
		t.Fatalf("Start reported %v while the program kept leaving", err)
	}
	if len(fake.Spawned) != 2 {
		t.Errorf("Start spawned %d times, want the retry as the second", len(fake.Spawned))
	}
	if want := 2*steam.LaunchCheck + steam.LaunchRetry; t2.total != want {
		t.Errorf("Start waited %v, want %v", t2.total, want)
	}
}

func TestStartSucceedsOnTheRetry(t *testing.T) {
	fake := &rt.Fake{Files: map[string]bool{root + `\steam.exe`: true}}
	client, _ := clientWith(fake)
	fake.OnSpawn = func(call rt.Launch) {
		if len(fake.Spawned) == 2 {
			fake.Procs = append(fake.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
		}
	}

	if err := client.Start(root); err != nil {
		t.Fatalf("Start reported %v although the retry lived", err)
	}
	if len(fake.Spawned) != 2 {
		t.Errorf("Start spawned %d times, want two attempts", len(fake.Spawned))
	}
}

func TestStartReportsAMissingProgram(t *testing.T) {
	fake := &rt.Fake{}
	client, t2 := clientWith(fake)

	if err := client.Start(root); !errors.Is(err, steam.ErrMissing) {
		t.Fatalf("Start reported %v without steam.exe, want ErrMissing", err)
	}
	if len(fake.Spawned) != 0 || t2.total != 0 {
		t.Errorf("Start tried without steam.exe: spawned %v after %v", fake.Spawned, t2.total)
	}
}

func TestStartNeedsADirectory(t *testing.T) {
	fake := &rt.Fake{Files: map[string]bool{"steam.exe": true}}
	client, _ := clientWith(fake)

	if err := client.Start(""); !errors.Is(err, steam.ErrMissing) {
		t.Fatalf("Start reported %v without a directory, want ErrMissing", err)
	}
	if len(fake.Spawned) != 0 {
		t.Errorf("Start spawned %v without a directory", fake.Spawned)
	}
}
