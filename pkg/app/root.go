package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/charmbracelet/log"
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

	withPeriodicTasks, ok := p.(module.WithPeriodicTasks)
	if ok {
		periodicTasks := withPeriodicTasks.PeriodicTasks()
		log.Info("Module supports periodic tasks", "module", fmt.Sprintf("%T", p), "count", len(periodicTasks))
		for i := range withPeriodicTasks.PeriodicTasks() {
			taskConfig := periodicTasks[i]
			log.Info("Registering task for module", "module", fmt.Sprintf("%T", p), "task", taskConfig.Name, "interval", taskConfig.Interval)
			task := module.NewTask(taskConfig.Name, taskConfig.Task, taskConfig.Interval)
			modules = append(modules, task)
		}
	}
}

func doRun(cmd *cobra.Command, args []string) {
	var config string
	var err error
	config, err = cmd.Flags().GetString("config")
	if err != nil {
		log.Error("Could not initialize config:", err)
		os.Exit(1)
	} else if config == "" {
		log.Error("--config cannot be an empty string")
		os.Exit(1)
	}

	viper.SetConfigFile(config)
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()

	if err != nil {
		log.Error("Could not read config:", err)
		os.Exit(1)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	ctx, cancelFunc := context.WithCancel(context.Background())
	errs, ctx := errgroup.WithContext(ctx)

	for i := range modules {
		module.MustConfigure(modules[i])
	}

	// Run init + first iteration of periodic tasks if any
	for i := range modules {
		if modules[i].HasInit() {
			err := modules[i].Init(ctx)
			if err != nil {
				log.Error("Failed to initialize module", "name", modules[i].GetName(), "error", err)
				os.Exit(1)
			}
		}

		withPeriodicTasks, ok := modules[i].(module.WithPeriodicTasks)
		if ok {
			periodicTasks := withPeriodicTasks.PeriodicTasks()
			for j := range periodicTasks {
				log.Info("Running initial iteration of periodic task for module", "name", modules[i].GetName(), "task", periodicTasks[j].Name)
				periodicTasks[j].Task()
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
			log.Error("Failed to run modules", "error", err)
			os.Exit(1)
		}
		mainDoneCh <- true
	}()

	select {
	case <-signalCh:
		log.Info("Got a signal, shutting down app")
		cancelFunc()

	case <-mainDoneCh:
		log.Info("Main completed, shutting down app")
		cancelFunc()
	}

	for i := range modules {
		if modules[i].HasCleanup() {
			modules[i].Cleanup(ctx)
		}
	}

	log.Info("All modules shut down, quitting")
}

func Execute() {
	err := RootCmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}
