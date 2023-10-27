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

var (
	logLevels map[string]logger.LogLevel
)

func init() {
	logLevels = map[string]logger.LogLevel{
		"silent": logger.Silent,
		"info":   logger.Info,
		"warn":   logger.Warn,
		"error":  logger.Error,
	}
}

type MigrationFunc = func(db *gorm.DB)

type Config struct {
	Host     string
	DBName   string
	Username string
	Password string
	Options  []string
	LogLevel string
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

	// already sanitized
	logLevel, _ := logLevels[dbConfig.LogLevel]
	p.db, err = gorm.Open(mysql.Open(connectionString), &gorm.Config{Logger: logger.Default.LogMode(logLevel)})

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
	options := viper.GetStringSlice(fmt.Sprintf("%s.options", configPrefix))
	logLevel := viper.GetString(fmt.Sprintf("%s.logLevel", configPrefix))

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

	if logLevel == "" {
		logLevel = "silent"
	} else if _, ok := logLevels[logLevel]; !ok {
		return nil, fmt.Errorf("Invalid configuration: invalid log level %s", logLevel)
	}

	return &Config{Host: host,
		DBName:   dbName,
		Username: username,
		Password: password,
		Options:  options,
		LogLevel: logLevel}, nil
}

func buildConnectionString(dbConfig *Config) string {
	opts := strings.Join(dbConfig.Options, "&")
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.DBName, opts)
}

func NewGORMDatabaseModule(name string, options *Options) GORMDatabaseModule {
	return GORMDatabaseModule{Base: module.Base{Name: name, IncludesInit: true}, options: options}
}
