package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dnikishov/microboiler/pkg/module"
)

var (
	RootCmd   *cobra.Command
	appConfig *Config
	modules   []module.Module
)

type Config struct {
	UseString  string
	DescString string
}

func Init(conf *Config) {
	appConfig = conf
	RootCmd = &cobra.Command{
		Use:   appConfig.UseString,
		Short: appConfig.DescString,
		Run:   doRun,
	}

	RootCmd.PersistentFlags().String("config", "", "Configuration file path")
	RootCmd.MarkPersistentFlagRequired("config")
}

func RegisterModule(p module.Module) {
	modules = append(modules, p)
}

func doRun(cmd *cobra.Command, args []string) {
	var config string
	var err error
	config, err = cmd.Flags().GetString("config")
	if err != nil {
		slog.Error("Could not initialize config:", err)
		os.Exit(1)
	} else if config == "" {
		slog.Error("--config cannot be an empty string")
		os.Exit(1)
	}

	viper.SetConfigFile(config)
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()

	if err != nil {
		slog.Error("Could not read config:", err)
		os.Exit(1)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	ctx := context.Background()
	errs, ctx := errgroup.WithContext(ctx)

	for i := range modules {
		if modules[i].HasInit() {
			err := modules[i].Init(ctx)
			if err != nil {
				slog.Error("Failed to initialize module", "name", modules[i].GetName(), "error", err)
				os.Exit(1)
			}
		}
	}

	for i := range modules {
		if modules[i].HasMain() {
			f := func() error {
				return modules[i].Main(ctx)
			}
			errs.Go(f)
			time.Sleep(1 * time.Second)
		}
	}

	mainDoneCh := make(chan bool, 1)
	go func() {
		err = errs.Wait()
		if err != nil {
			slog.Error("Failed to run modules", "error", err)
			os.Exit(1)
		}
		mainDoneCh <- true
	}()

	select {
	case <-signalCh:
		slog.Info("Got a signal, shutting down app")
	case <-mainDoneCh:
		slog.Info("Main completed, shutting down app")
	}

	for i := range modules {
		if modules[i].HasCleanup() {
			modules[i].Cleanup(ctx)
		}
	}

	slog.Info("All modules shut down, quitting")
}

func Execute() {
	err := RootCmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}
