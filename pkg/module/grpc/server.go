package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

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
		return errors.New("GRPC: listenAddress must be specified")
	}

	p.ctx = ctx
	p.server = grpc.NewServer()

	slog.Info("GRPC server module initialized")

	p.registerServices()
	p.listenAddress = listenAddress

	return nil
}

func (p *GRPCServerModule) Main(_ context.Context) error {
	return p.startServer()
}

func (p *GRPCServerModule) Cleanup(_ context.Context) {
	slog.Info("Stopping GRPC server")
	p.server.Stop()
}

func (p *GRPCServerModule) startServer() error {
	serveErrorCh := make(chan error)

	go p.doServe(serveErrorCh)

	select {
	case err := <-serveErrorCh:
		return err

	// probably sub-optimal but good enough at this point
	case <-time.After(1 * time.Second):
		return nil
	}
}

func (p *GRPCServerModule) doServe(serveErrorCh chan error) {
	listener, err := net.Listen("tcp", p.listenAddress)
	if err != nil {
		serveErrorCh <- errors.New(fmt.Sprintf("Could not initialize listener on %s: %s", p.listenAddress, err))
	}
	err = p.server.Serve(listener)
	if err != nil {
		serveErrorCh <- errors.New(fmt.Sprintf("Could not start GRPC server: %s", err))
	}

	serveErrorCh <- nil
}

func (p *GRPCServerModule) registerServices() {
	for _, entry := range p.options.ServiceRegistry {
		p.server.RegisterService(&entry.ServiceDesc, entry.Service)
	}
}

func NewGRPCServerModule(name string, options *Options) GRPCServerModule {
	mName := fmt.Sprintf("GRPC server %s", name)
	return GRPCServerModule{Base: module.Base{Name: mName, IncludesInit: true, IncludesCleanup: true}, options: options}
}
