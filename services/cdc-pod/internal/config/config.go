package config

import (
	"fmt"
	"net/url"

	"github.com/caarlos0/env/v11"
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

// StandardConnString returns a PostgreSQL connection string for regular queries.
func (c *Config) StandardConnString() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   fmt.Sprintf("%s:%d", c.DBHost, c.DBPort),
		Path:   c.DBName,
	}
	return u.String()
}

// ReplicationConnString returns a PostgreSQL connection string for replication protocol.
func (c *Config) ReplicationConnString() string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPassword),
		Host:     fmt.Sprintf("%s:%d", c.DBHost, c.DBPort),
		Path:     c.DBName,
		RawQuery: "replication=database",
	}
	return u.String()
}
