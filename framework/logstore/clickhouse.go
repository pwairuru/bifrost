package logstore

import (
	"context"
	"fmt"

	"github.com/maximhq/bifrost/core/schemas"

	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

type ClickHouseConfig struct {
	Host         *schemas.EnvVar `json:"host"`
	Port         *schemas.EnvVar `json:"port"`
	DBName       *schemas.EnvVar `json:"db_name"`
	User         *schemas.EnvVar `json:"user"`
	Password     *schemas.EnvVar `json:"password"`
	MaxIdleConns int             `json:"max_idle_conns"`
	MaxOpenConns int             `json:"max_open_conns"`
}

func newClickHouseLogStore(ctx context.Context, config *ClickHouseConfig, logger schemas.Logger) (LogStore, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.Host == nil || config.Host.GetValue() == "" {
		return nil, fmt.Errorf("clickhouse host is required")
	}
	if config.Port == nil || config.Port.GetValue() == "" {
		return nil, fmt.Errorf("clickhouse port is required")
	}
	if config.User == nil || config.User.GetValue() == "" {
		return nil, fmt.Errorf("clickhouse user is required")
	}
	if config.DBName == nil || config.DBName.GetValue() == "" {
		return nil, fmt.Errorf("clickhouse db name is required")
	}

	password := ""
	if config.Password != nil {
		password = config.Password.GetValue()
	}
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%s/%s",
		config.User.GetValue(), password,
		config.Host.GetValue(), config.Port.GetValue(), config.DBName.GetValue())

	db, err := gorm.Open(clickhouse.Open(dsn), &gorm.Config{
		Logger: newGormLogger(logger),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	maxIdleConns := config.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 5
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)

	maxOpenConns := config.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 50
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)

	d := &RDBLogStore{db: db, logger: logger}
	if err := triggerClickhouseMigrations(ctx, db); err != nil {
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			sqlDB.Close()
		}
		return nil, err
	}
	return d, nil
}
