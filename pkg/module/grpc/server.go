package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/dnikishov/microboiler/pkg/module"
)

type RegistryEntry struct {
	ServiceDesc grpc.ServiceDesc
	Service     interface{}
}

type Options struct {
	ServiceRegistry []RegistryEntry
}

type GRPCServerModule struct {
	module.Base
	server        *grpc.Server
	options       *Options
	ctx           context.Context
	listenAddress string
}

func (p *GRPCServerModule) Configure() error {
	configPrefix := fmt.Sprintf("grpc-%s", p.GetName())
	listenAddress := viper.GetString(fmt.Sprintf("%s.listenAddress", configPrefix))

	if listenAddress == "" {
		return fmt.Errorf("invalid configuration: %s.listenAddress is not set", configPrefix)
	}

	p.listenAddress = listenAddress

	for _, entry := range p.options.ServiceRegistry {
		configurableSvc, ok := entry.Service.(module.Configurable)
		if ok {
			slog.Info("Configuring GRPC service", "name", p.GetName(), "service", fmt.Sprintf("%T", entry.Service))
			err := configurableSvc.Configure()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *GRPCServerModule) PeriodicTasks() []*module.TaskConfig {
	tasks := make([]*module.TaskConfig, 0)

	for i := range p.options.ServiceRegistry {
		entry := p.options.ServiceRegistry[i]
		withPeriodicTasksSvc, ok := entry.Service.(module.WithPeriodicTasks)
		if ok {
			periodicTasks := withPeriodicTasksSvc.PeriodicTasks()
			slog.Info("GRPC service supports periodic tasks", "service", fmt.Sprintf("%T", entry.Service), "count", len(periodicTasks))
			tasks = append(tasks, periodicTasks...)
		}
	}

	return tasks
}

func (p *GRPCServerModule) Init(ctx context.Context) error {
	p.ctx = ctx
	p.server = grpc.NewServer()
	p.registerServices()

	slog.Info("GRPC server initialized", "name", p.GetName(), "address", p.listenAddress)

	return nil
}

func (p *GRPCServerModule) Main(_ context.Context) error {
	listener, err := net.Listen("tcp", p.listenAddress)
	if err != nil {
		return err
	}
	err = p.server.Serve(listener)
	if err != nil {
		return err
	}
	return nil
}

func (p *GRPCServerModule) Cleanup(_ context.Context) {
	slog.Info("Stopping GRPC server", "name", p.GetName())
	p.server.Stop()
}

func (p *GRPCServerModule) registerServices() {
	for _, entry := range p.options.ServiceRegistry {
		p.server.RegisterService(&entry.ServiceDesc, entry.Service)
	}
}

func NewGRPCServerModule(name string, options *Options) *GRPCServerModule {
	return &GRPCServerModule{Base: module.Base{Name: name, IncludesInit: true, IncludesCleanup: true, IncludesMain: true}, options: options}
}
