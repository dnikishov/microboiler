package module

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Configurable interface {
	Configure() error
}

type WithPeriodicTasks interface {
	PeriodicTasks() []*TaskConfig
}

type TaskConfig struct {
	Name     string
	Task     TaskFunc
	Interval time.Duration
}

type Module interface {
	GetName() string

	HasInit() bool
	HasCleanup() bool
	HasMain() bool

	Configure() error
	Init(ctx context.Context) error
	Main(ctx context.Context) error
	Cleanup(ctx context.Context)
}

type Base struct {
	Name            string
	IncludesInit    bool
	IncludesCleanup bool
	IncludesMain    bool
}

func (m Base) GetName() string {
	return m.Name
}

func (m Base) HasInit() bool {
	return m.IncludesInit
}

func (m Base) HasCleanup() bool {
	return m.IncludesCleanup
}

func (m Base) HasMain() bool {
	return m.IncludesMain
}

func (m *Base) Configure() error {
	return nil
}

func (m Base) Init(_ context.Context) error {
	panic(fmt.Sprintf("Init is not implemented in %s", m.GetName()))
}

func (m Base) Main(_ context.Context) error {
	panic(fmt.Sprintf("Main is not implemented in %s", m.GetName()))
}

func (m Base) Cleanup(_ context.Context) {
	panic(fmt.Sprintf("Cleanup is not implemented in %s", m.GetName()))
}

type TaskFunc = func()

type Task struct {
	Base
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

	slog.Info("Periodic task stopped", "name", p.GetName())
	return nil
}

func MustConfigure(mod Module) {
	err := mod.Configure()
	if err != nil {
		panic(fmt.Sprintf("Configuration failed for %T %s: %s", mod, mod.GetName(), err))
	}
}

func NewTask(name string, task TaskFunc, interval time.Duration) *Task {
	return &Task{Base: Base{Name: name, IncludesMain: true}, task: task, interval: interval}
}
