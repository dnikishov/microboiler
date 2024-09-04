package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/charmbracelet/log"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

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
	server          *grpc.Server
	options         *Options
	ctx             context.Context
	listenAddress   string
	exportMetrics   bool
	keepaliveParams keepalive.ServerParameters

	Metrics *grpcprom.ServerMetrics
}

func (p *GRPCServerModule) parseKeepaliveParams() {
	configPrefix := fmt.Sprintf("grpc-%s", p.GetName())

	p.keepaliveParams.MaxConnectionIdle = viper.GetDuration(fmt.Sprintf("%s.keepalive.maxConnectionIdle", configPrefix))
	p.keepaliveParams.MaxConnectionAge = viper.GetDuration(fmt.Sprintf("%s.keepalive.maxConnectionAge", configPrefix))
	p.keepaliveParams.MaxConnectionAgeGrace = viper.GetDuration(fmt.Sprintf("%s.keepalive.maxConnectionAgeGrace", configPrefix))
	p.keepaliveParams.Time = viper.GetDuration(fmt.Sprintf("%s.keepalive.time", configPrefix))
	p.keepaliveParams.Timeout = viper.GetDuration(fmt.Sprintf("%s.keepalive.timeout", configPrefix))
}

func (p *GRPCServerModule) Configure() error {
	configPrefix := fmt.Sprintf("grpc-%s", p.GetName())
	listenAddress := viper.GetString(fmt.Sprintf("%s.listenAddress", configPrefix))
	exportMetrics := viper.GetBool(fmt.Sprintf("%s.exportMetrics", configPrefix))

	if listenAddress == "" {
		return fmt.Errorf("invalid configuration: %s.listenAddress is not set", configPrefix)
	}

	p.listenAddress = listenAddress
	p.exportMetrics = exportMetrics

	p.parseKeepaliveParams()

	for _, entry := range p.options.ServiceRegistry {
		configurableSvc, ok := entry.Service.(module.Configurable)
		if ok {
			log.Info("Configuring GRPC service", "name", p.GetName(), "service", fmt.Sprintf("%T", entry.Service))
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
			log.Info("GRPC service supports periodic tasks", "service", fmt.Sprintf("%T", entry.Service), "count", len(periodicTasks))
			tasks = append(tasks, periodicTasks...)
		}
	}

	return tasks
}

func (p *GRPCServerModule) Init(ctx context.Context) error {
	p.ctx = ctx
	serverOptions := []grpc.ServerOption{grpc.KeepaliveParams(p.keepaliveParams)}
	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	streamInterceptors := []grpc.StreamServerInterceptor{}

	if p.exportMetrics {
		unaryInterceptors = append(
			unaryInterceptors,
			p.Metrics.UnaryServerInterceptor(),
		)

		streamInterceptors = append(
			streamInterceptors,
			p.Metrics.StreamServerInterceptor(),
		)
	}

	if len(unaryInterceptors) > 0 {
		unaryInterceptorOpt := grpc.ChainUnaryInterceptor(unaryInterceptors...)
		serverOptions = append(serverOptions, unaryInterceptorOpt)
	}

	if len(streamInterceptors) > 0 {
		streamInterceptorOpt := grpc.ChainStreamInterceptor(streamInterceptors...)
		serverOptions = append(serverOptions, streamInterceptorOpt)
	}

	p.server = grpc.NewServer(serverOptions...)
	p.registerServices()

	log.Info("GRPC server initialized", "name", p.GetName(), "address", p.listenAddress)

	return nil
}

func (p *GRPCServerModule) Main(_ context.Context) error {
	listener, err := net.Listen("tcp", p.listenAddress)
	if err != nil {
		return err
	}
	log.Info("Starting GRPC server", "name", p.GetName(), "address", p.listenAddress)
	err = p.server.Serve(listener)
	if err != nil {
		return err
	}
	return nil
}

func (p *GRPCServerModule) Cleanup(_ context.Context) {
	log.Info("Stopping GRPC server", "name", p.GetName())
	p.server.Stop()
}

func (p *GRPCServerModule) registerServices() {
	for _, entry := range p.options.ServiceRegistry {
		p.server.RegisterService(&entry.ServiceDesc, entry.Service)
	}
}

func NewGRPCServerModule(name string, options *Options) *GRPCServerModule {
	metrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerCounterOptions(grpcprom.WithConstLabels(prometheus.Labels{"app": name})),
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	return &GRPCServerModule{Base: module.Base{Name: name, IncludesInit: true, IncludesCleanup: true, IncludesMain: true}, options: options, Metrics: metrics}
}
