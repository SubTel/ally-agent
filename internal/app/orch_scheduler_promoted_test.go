package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

type recordingTaskSink struct{ names []string }

func (s *recordingTaskSink) Emit(name string, _ any) { s.names = append(s.names, name) }

// A scheduled command that timed out into a background service blocks its own
// next run: the old flow cleared Running and fired again, giving two live
// processes writing the same output or fighting over the same port.
func TestScheduledCommandSkipsWhilePromotedServiceLives(t *testing.T) {
	t.Parallel()
	a := newIsolatedTestApp(t)
	a.initialized = true
	a.ctx = context.Background()
	workspace := t.TempDir()
	a.config.Workspace = workspace

	manager := &scheduledTaskManager{
		app:       a,
		events:    &recordingTaskSink{},
		cron:      cron.New(),
		path:      filepath.Join(t.TempDir(), "scheduled_tasks.json"),
		tasks:     map[string]*ScheduledTask{},
		entries:   map[string]cron.EntryID{},
		timers:    map[string]*time.Timer{},
		schedules: map[string]cron.Schedule{},
		cancels:   map[string]context.CancelFunc{},
		runSem:    make(chan struct{}, 1),
		promoted:  map[string]string{},
	}
	task := &ScheduledTask{ID: "task-promoted", Name: "promoted", Command: "exit 0", Workspace: workspace, MaxSteps: 1}
	manager.tasks[task.ID] = task
	manager.promoted[task.ID] = "svc-live"
	a.services["svc-live"] = &managedService{info: ServiceInfo{ID: "svc-live", Status: "running", Cwd: workspace}}

	manager.trigger(task.ID)

	if task.LastStatus != "skipped" || !strings.Contains(task.LastError, "background service") {
		t.Fatalf("the live service did not block the next run: status=%q error=%q", task.LastStatus, task.LastError)
	}
	if task.RunCount != 0 {
		t.Fatalf("the skipped run still executed the command: %d", task.RunCount)
	}

	// Once the service is gone the same task runs again normally.
	a.services["svc-live"].mu.Lock()
	a.services["svc-live"].info.Status = "stopped"
	a.services["svc-live"].mu.Unlock()
	manager.trigger(task.ID)
	manager.workers.Wait()
	if task.LastStatus != "completed" || task.RunCount != 1 {
		t.Fatalf("the run after the service exited did not complete: status=%q count=%d error=%q", task.LastStatus, task.RunCount, task.LastError)
	}
}
