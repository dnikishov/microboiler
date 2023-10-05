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
)

type RegistryEntry struct {
	ServiceDesc grpc.ServiceDesc
	Service     interface{}
}

type Options struct {
	ServiceRegistry []RegistryEntry
}

type GRPCServerModule struct {
	server  *grpc.Server
	options *Options
	ctx     context.Context
}

func (p *GRPCServerModule) Start(ctx context.Context) error {
	listenAddress := viper.GetString("grpc.listenAddress")

	if listenAddress == "" {
		return errors.New("GRPC: listenAddress must be specified")
	}

	p.ctx = ctx
	p.server = grpc.NewServer()

	slog.Info("GRPC server module initialized")

	p.registerServices()

	return p.startServer(listenAddress)
}

func (p *GRPCServerModule) Cleanup() {
	slog.Info("Stopping GRPC server")
	p.server.Stop()
}

func (p *GRPCServerModule) startServer(listenAddress string) error {
	serveErrorCh := make(chan error)

	go p.doServe(listenAddress, serveErrorCh)

	select {
	case err := <-serveErrorCh:
		return err

	// probably sub-optimal but good enough at this point
	case <-time.After(1 * time.Second):
		return nil
	}
}

func (p *GRPCServerModule) doServe(listenAddress string, serveErrorCh chan error) {
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		serveErrorCh <- errors.New(fmt.Sprintf("Could not initialize listener on %s: %s", listenAddress, err))
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

func NewGRPCServerModule(options *Options) GRPCServerModule {
	return GRPCServerModule{options: options}
}
