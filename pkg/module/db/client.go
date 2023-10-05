package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dnikishov/microboiler/pkg/module"
)

type MigrationFunc = func(db *gorm.DB)

type Config struct {
	Host     string
	DBName   string
	Username string
	Password string
	Options  map[string]string
}

type Options struct {
	Migrations []MigrationFunc
}

type GORMDatabaseModule struct {
	module.Base
	db      *gorm.DB
	options *Options
}

func (p *GORMDatabaseModule) Init(_ context.Context) error {
	dbConfig, err := p.loadConfigFromViper()

	if err != nil {
		return err
	}

	connectionString := buildConnectionString(dbConfig)

	p.db, err = gorm.Open(mysql.Open(connectionString), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})

	if err != nil {
		return fmt.Errorf("Failed to initialize DB module: %s", err)
	}

	for _, migrationFunc := range p.options.Migrations {
		migrationFunc(p.db)
	}

	slog.Info("GORM database module initialized", "name", p.GetName())

	return nil
}

func (p *GORMDatabaseModule) GetDB() *gorm.DB {
	return p.db
}

func (p *GORMDatabaseModule) loadConfigFromViper() (*Config, error) {
	configPrefix := fmt.Sprintf("gorm-%s", p.GetName())

	host := viper.GetString(fmt.Sprintf("%s.host", configPrefix))
	dbName := viper.GetString(fmt.Sprintf("%s.name", configPrefix))
	username := viper.GetString(fmt.Sprintf("%s.username", configPrefix))
	password := viper.GetString(fmt.Sprintf("%s.password", configPrefix))
	options := viper.GetStringMapString(fmt.Sprintf("%s.options", configPrefix))

	if host == "" {
		return nil, fmt.Errorf("Invalid configuration: %s.host is not set", configPrefix)
	}

	if dbName == "" {
		return nil, fmt.Errorf("Invalid configuration: %s.dbName is not set", configPrefix)
	}

	if username == "" {
		return nil, fmt.Errorf("Invalid configuration: %s.username is not set", configPrefix)
	}

	if password == "" {
		return nil, fmt.Errorf("Invalid configuration: %s.password is not set", configPrefix)
	}

	return &Config{Host: host,
		DBName:   dbName,
		Username: username,
		Password: password,
		Options:  options}, nil
}

func buildConnectionString(dbConfig *Config) string {
	var optsList []string
	for opt, val := range dbConfig.Options {
		optsList = append(optsList, fmt.Sprintf("%s=%s", opt, val))
	}
	opts := strings.Join(optsList, "&")
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.DBName, opts)
}

func NewGORMDatabaseModule(name string, options *Options) GORMDatabaseModule {
	return GORMDatabaseModule{Base: module.Base{Name: name, IncludesInit: true}, options: options}
}
