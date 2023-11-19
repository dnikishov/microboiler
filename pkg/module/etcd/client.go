package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/viper"
	client "go.etcd.io/etcd/client/v3"

	"github.com/dnikishov/microboiler/pkg/module"
)

type EtcdClientModule struct {
	module.Base
	client *client.Client
	cfg client.Config
}

func (p *EtcdClientModule) Configure() error {
	configPrefix := fmt.Sprintf("etcd-%s", p.GetName())
	endpoints := viper.GetStringSlice(fmt.Sprintf("%s.endpoints", configPrefix))

	if len(endpoints) == 0 {
		return fmt.Errorf("Invalid configuration: missing `endpoints` configuration")
	}

	p.cfg = client.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	}

	return nil
}

func (p *EtcdClientModule) Init(_ context.Context) error {
	var err error
	p.client, err = client.New(p.cfg)
	if err != nil {
		slog.Error("Failed to initialize Etcd client", "name", p.GetName(), "error", err)
		return err
	}

	return nil
}

func (p *EtcdClientModule) Cleanup(_ context.Context) {
	if p.client != nil {
		p.client.Close()
	}
}

func NewEtcdClientModule(name string) *EtcdClientModule {
	return &EtcdClientModule{Base: module.Base{Name: name, IncludesInit: true}}
}

func (p *EtcdClientModule) GetClient() *client.Client {
	return p.client
}
