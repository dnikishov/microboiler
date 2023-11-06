package prometheus

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/spf13/viper"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/dnikishov/microboiler/pkg/module"
)

type CollectorDefinition struct {
	Name      string
	Collector prometheus.Collector
}

type Config struct {
	MetricsPath   string
	ListenAddress string
	MaxRequests   int
}

type Options struct {
	TitleString          string
	CollectorDefinitions []CollectorDefinition
}

type PrometheusExporterModule struct {
	module.Base
	options *Options
	config  *Config

	registry *prometheus.Registry
	serveMux *http.ServeMux
	server   *http.Server
}

func (p *PrometheusExporterModule) Init(_ context.Context) error {
	err := p.loadConfigFromViper()

	if err != nil {
		return err
	}

	p.registry = prometheus.NewRegistry()

	for _, def := range p.options.CollectorDefinitions {
		slog.Info("Registering collector", "name", p.GetName(), "collector_name", def.Name)
		p.registry.MustRegister(def.Collector)
	}

	handler := promhttp.HandlerFor(
		prometheus.Gatherers{p.registry},
		promhttp.HandlerOpts{
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: p.config.MaxRequests,
			Registry:            p.registry,
		},
	)

	p.serveMux = http.NewServeMux()
	p.serveMux.Handle(p.config.MetricsPath, handler)

	indexBody := fmt.Sprintf(
		`<html>
		<head><title>%s</title></head>
		<body>
		<h1>%s</h1>
		<p><a href="%s">Metrics</a></p>
		</body>
	</html>`, p.options.TitleString, p.options.TitleString, p.config.MetricsPath,
	)
	p.serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(indexBody))
	})

	return nil
}

func (p *PrometheusExporterModule) Main(_ context.Context) error {
	slog.Info("Starting prometheus exporter", "name", p.GetName(), "address", p.config.ListenAddress)
	p.server = &http.Server{Addr: p.config.ListenAddress, Handler: p.serveMux}
	err := p.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	slog.Info("Prometheus exporter stopped", "name", p.GetName(), "address", p.config.ListenAddress)
	return nil
}

func (p *PrometheusExporterModule) loadConfigFromViper() error {
	configPrefix := fmt.Sprintf("prometheus-exporter-%s", p.GetName())

	metricsPath := viper.GetString(fmt.Sprintf("%s.metrics_path", configPrefix))
	listenAddress := viper.GetString(fmt.Sprintf("%s.listen_address", configPrefix))
	maxRequests := viper.GetInt(fmt.Sprintf("%s.max_requests", configPrefix))

	if metricsPath == "" {
		metricsPath = "/metrics"
	}

	if listenAddress == "" {
		listenAddress = ":9300"
	}

	if maxRequests == 0 {
		maxRequests = 1
	} else if maxRequests < 0 {
		return fmt.Errorf("Invalid configuration: %s.max_requests must be greater than 0", configPrefix)
	}

	p.config = &Config{
		MetricsPath:   metricsPath,
		ListenAddress: listenAddress,
		MaxRequests:   maxRequests,
	}

	return nil
}

func (p *PrometheusExporterModule) Cleanup(ctx context.Context) {
	slog.Info("Stopping prometheus exporter", "name", p.GetName())
	p.server.Shutdown(ctx)
}

func NewPrometheusExporterModule(name string, options *Options) PrometheusExporterModule {
	return PrometheusExporterModule{Base: module.Base{Name: name, IncludesInit: true, IncludesMain: true, IncludesCleanup: true}, options: options}
}
