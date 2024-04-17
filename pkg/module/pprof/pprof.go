package pprof

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"

	"github.com/dnikishov/microboiler/pkg/module"
	"github.com/spf13/viper"
)

type Config struct {
	ListenAddress string
}

type PprofModule struct {
	module.Base
	config *Config

	server   *http.Server
	serveMux *http.ServeMux
}

func (p *PprofModule) Init(_ context.Context) error {
	p.serveMux = http.NewServeMux()
	p.serveMux.HandleFunc("/debug/pprof/", pprof.Index)
	p.serveMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	p.serveMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	p.serveMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	p.serveMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return nil
}

func (p *PprofModule) Main(_ context.Context) error {
	slog.Info("Starting pprof server", "name", p.GetName(), "address", p.config.ListenAddress)
	p.server = &http.Server{Addr: p.config.ListenAddress, Handler: p.serveMux}
	err := p.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	slog.Info("Pprof server stopped", "name", p.GetName(), "address", p.config.ListenAddress)
	return nil
}

func (p *PprofModule) Configure() error {
	configPrefix := fmt.Sprintf("pprof-%s", p.GetName())

	listenAddress := viper.GetString(fmt.Sprintf("%s.listen_address", configPrefix))

	if listenAddress == "" {
		listenAddress = "localhost:8080"
	}

	p.config = &Config{
		ListenAddress: listenAddress,
	}

	return nil
}

func (p *PprofModule) Cleanup(ctx context.Context) {
	slog.Info("Stopping pprof server", "name", p.GetName())
	p.server.Shutdown(ctx)
}

func NewPprofModule(name string) *PprofModule {
	return &PprofModule{Base: module.Base{Name: name, IncludesInit: true, IncludesMain: true, IncludesCleanup: true}}
}
