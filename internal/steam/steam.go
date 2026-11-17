package steam

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"opensteamtool-gui-mygo/internal/rt"
)

const (
	Exe     = "steam.exe"
	Service = "steamservice.exe"

	shutdownArgument = "-shutdown"

	PollInterval   = 2 * time.Second
	GracefulBudget = 10 * time.Second
	ForceBudget    = 5 * time.Second
	LaunchCheck    = 2 * time.Second
	LaunchRetry    = 1 * time.Second
	Step           = 500 * time.Millisecond
)

var (
	ErrMissing    = errors.New("steam.exe is not in the Steam directory")
	ErrNotStarted = errors.New("a started Steam did not stay up")
	ErrNotStopped = errors.New("the Steam process group did not empty")
)

type Client struct {
	host  rt.Runtime
	sleep func(time.Duration)
}

func New(host rt.Runtime, sleep func(time.Duration)) *Client {
	return &Client{host: host, sleep: sleep}
}

func Running(host rt.Runtime, root string) (bool, error) {
	processes, err := group(host, root)
	if err != nil {
		return false, err
	}
	return len(processes) > 0, nil
}

func (c *Client) Stop(root string) error {
	processes, err := group(c.host, root)
	if err != nil {
		return fmt.Errorf("list processes: %w", err)
	}
	if len(processes) == 0 {
		return nil
	}
	var helper int
	if main, ok := mainOf(processes, root); ok && c.host.IsFile(main.Exe) {
		helper, err = c.host.Spawn(main.Exe, root, shutdownArgument)
		if err == nil && empty(c.host, root, c.sleep, GracefulBudget) {
			return nil
		}
	}
	failure := c.force(root, helper)
	if empty(c.host, root, c.sleep, ForceBudget) {
		return nil
	}
	if failure != nil {
		return failure
	}
	return ErrNotStopped
}

func (c *Client) Start(root string) error {
	if root == "" {
		return ErrMissing
	}
	exe := filepath.Join(root, Exe)
	if !c.host.IsFile(exe) {
		return ErrMissing
	}
	for attempt := 0; attempt < 2; attempt++ {
		pid, err := c.host.Spawn(exe, root)
		if err == nil {
			c.sleep(LaunchCheck)
			if alive(c.host, root, pid) {
				return nil
			}
		}
		if attempt == 0 {
			c.sleep(LaunchRetry)
		}
	}
	return ErrNotStarted
}

func (c *Client) force(root string, helper int) error {
	processes, err := group(c.host, root)
	if err != nil {
		return fmt.Errorf("list processes: %w", err)
	}
	var failure error
	for _, process := range processes {
		if process.PID == helper {
			continue
		}
		if err := c.host.Terminate(process.PID); err != nil {
			failure = errors.Join(failure, fmt.Errorf("terminate %d: %w", process.PID, err))
		}
	}
	return failure
}

func group(host rt.Runtime, root string) ([]rt.Process, error) {
	all, err := host.Processes()
	if err != nil {
		return nil, err
	}
	var processes []rt.Process
	for _, process := range all {
		if !under(process.Exe, root) || strings.EqualFold(filepath.Base(process.Exe), Service) {
			continue
		}
		processes = append(processes, process)
	}
	return processes, nil
}

func mainOf(processes []rt.Process, root string) (rt.Process, bool) {
	main := filepath.Join(root, Exe)
	for _, process := range processes {
		if strings.EqualFold(process.Exe, main) {
			return process, true
		}
	}
	return rt.Process{}, false
}

func alive(host rt.Runtime, root string, pid int) bool {
	processes, err := group(host, root)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(processes, func(process rt.Process) bool { return process.PID == pid })
}

func empty(host rt.Runtime, root string, sleep func(time.Duration), budget time.Duration) bool {
	for waited := time.Duration(0); waited < budget; waited += Step {
		if emptyOf(host, root) {
			return true
		}
		sleep(Step)
	}
	return emptyOf(host, root)
}

func emptyOf(host rt.Runtime, root string) bool {
	processes, err := group(host, root)
	if err != nil {
		return false
	}
	return len(processes) == 0
}
func under(exe string, root string) bool {
	if root == "" {
		return false
	}
	exe = strings.ToLower(strings.ReplaceAll(exe, "/", `\`))
	root = strings.TrimSuffix(strings.ToLower(strings.ReplaceAll(root, "/", `\`)), `\`)
	return len(exe) > len(root) && strings.HasPrefix(exe, root) && exe[len(root)] == '\\'
}
