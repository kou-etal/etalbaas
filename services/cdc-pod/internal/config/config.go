package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Config holds all CDC Pod configuration, loaded from environment variables.
type Config struct {
	ProjectID       string `env:"CDC_PROJECT_ID,required"`
	SlotName        string `env:"CDC_SLOT_NAME,required"`
	NATSURL         string `env:"CDC_NATS_URL,required"`
	SubjectPrefix   string `env:"CDC_NATS_SUBJECT_PREFIX" envDefault:"events.database"`
	PublicationName string `env:"CDC_PUBLICATION_NAME"     envDefault:""`

	DBHost     string `env:"CDC_DB_HOST,required"`
	DBUser     string `env:"CDC_DB_USER,required"`
	DBPassword string `env:"CDC_DB_PASSWORD,required"`
	DBName     string `env:"CDC_DB_NAME,required"`
	DBPort     int    `env:"CDC_DB_PORT"      envDefault:"5432"`

	HealthPort             int `env:"CDC_HEALTH_PORT"              envDefault:"8081"`
	StandbyStatusIntervalS int `env:"CDC_STANDBY_STATUS_INTERVAL_S" envDefault:"10"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// EffectivePublicationName returns the publication name, defaulting to "cdc_{project_id}".
func (c *Config) EffectivePublicationName() string {
	if c.PublicationName != "" {
		return c.PublicationName
	}
	return "cdc_" + c.ProjectID
}

// StandardConnConfig returns a pgx connection config for regular queries.
// Uses programmatic config to prevent passwords from leaking into error messages.
func (c *Config) StandardConnConfig() (*pgx.ConnConfig, error) {
	cfg, err := pgx.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("parse empty pgx config: %w", err)
	}
	cfg.Host = c.DBHost
	cfg.Port = uint16(c.DBPort)
	cfg.User = c.DBUser
	cfg.Password = c.DBPassword
	cfg.Database = c.DBName
	return cfg, nil
}

// ReplicationConnConfig returns a pgconn connection config for replication protocol.
// Uses programmatic config to prevent passwords from leaking into error messages.
func (c *Config) ReplicationConnConfig() (*pgconn.Config, error) {
	cfg, err := pgconn.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("parse empty pgconn config: %w", err)
	}
	cfg.Host = c.DBHost
	cfg.Port = uint16(c.DBPort)
	cfg.User = c.DBUser
	cfg.Password = c.DBPassword
	cfg.Database = c.DBName
	cfg.RuntimeParams["replication"] = "database"
	return cfg, nil
}
