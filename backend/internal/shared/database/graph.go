package database

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type GraphDBConfig struct {
	URI         string
	Username    string
	Password    string
	Database    string
	MaxConnPool int
}

func NewGraphConnection(ctx context.Context, config GraphDBConfig) (neo4j.DriverWithContext, error) {
	// Determine auth: no-auth when username is empty, basic auth otherwise
	var auth neo4j.AuthToken
	if config.Username != "" {
		auth = neo4j.BasicAuth(config.Username, config.Password, "")
	} else {
		auth = neo4j.NoAuth()
	}

	driver, err := neo4j.NewDriverWithContext(config.URI, auth, func(c *neo4j.Config) {
		c.MaxConnectionPoolSize = config.MaxConnPool
		c.ConnectionAcquisitionTimeout = 30 * time.Second
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create graph database driver: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify graph database connectivity: %w", err)
	}

	return driver, nil
}

func DisconnectGraph(ctx context.Context, driver neo4j.DriverWithContext) error {
	return driver.Close(ctx)
}
