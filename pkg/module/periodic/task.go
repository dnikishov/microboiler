package periodic

import (
	"context"
	"log/slog"
	"time"

	"github.com/dnikishov/microboiler/pkg/module"
)

type TaskFunc = func()

type Task struct {
	module.Base
	interval time.Duration
	task     TaskFunc
}

func (p *Task) Main(ctx context.Context) error {
	slog.Info("Starting periodic task", "name", p.GetName())

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

mainLoop:
	for {
		select {
		case <-ctx.Done():
			break mainLoop
		case <-ticker.C:
			p.task()
		}
	}

	return nil
}

func NewTask(name string, task TaskFunc, interval time.Duration) Task {
	return Task{Base: module.Base{Name: name, IncludesMain: true}, task: task, interval: interval}
}
