package db

import (
	"context"
	"errors"
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
	dbConfig, err := loadConfigFromViper()

	if err != nil {
		return err
	}

	connectionString := buildConnectionString(dbConfig)

	p.db, err = gorm.Open(mysql.Open(connectionString), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})

	if err != nil {
		return errors.New(fmt.Sprintf("Failed to initialize DB module: %s", err))
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

func loadConfigFromViper() (*Config, error) {
	host := viper.GetString("db.host")
	dbName := viper.GetString("db.name")
	username := viper.GetString("db.username")
	password := viper.GetString("db.password")
	options := viper.GetStringMapString("db.options")

	if host == "" {
		return nil, errors.New("DB config: DB hostname must be specified")
	}

	if dbName == "" {
		return nil, errors.New("DB config: DB name must be specified")
	}

	if username == "" {
		return nil, errors.New("DB config: username must be specified")
	}

	if password == "" {
		return nil, errors.New("DB config: password must be specified")
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
