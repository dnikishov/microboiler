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

func (p *GRPCServerModule) Init(ctx context.Context) error {
	listenAddress := viper.GetString("grpc.listenAddress")

	if listenAddress == "" {
		return fmt.Errorf("GRPC: listenAddress must be specified")
	}

	p.ctx = ctx
	p.server = grpc.NewServer()

	slog.Info("GRPC server initialized", "name", p.GetName())

	p.registerServices()
	p.listenAddress = listenAddress

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

func NewGRPCServerModule(name string, options *Options) GRPCServerModule {
	return GRPCServerModule{Base: module.Base{Name: name, IncludesInit: true, IncludesCleanup: true}, options: options}
}
