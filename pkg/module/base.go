package module

import (
	"context"
	"fmt"
)

type Module interface {
	GetName() string

	HasInit() bool
	HasCleanup() bool
	HasMain() bool

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

func (m Base) Init(_ context.Context) error {
	panic(fmt.Sprintf("Init is not implemented in %s", m.GetName()))
}

func (m Base) Main(_ context.Context) error {
	panic(fmt.Sprintf("Main is not implemented in %s", m.GetName()))
}

func (m Base) Cleanup(_ context.Context) {
	panic(fmt.Sprintf("Cleanup is not implemented in %s", m.GetName()))
}
