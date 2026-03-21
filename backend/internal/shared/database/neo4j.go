package database

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jConfig struct {
	URI         string
	Username    string
	Password    string
	Database    string
	MaxConnPool int
}

func NewNeo4jConnection(ctx context.Context, config Neo4jConfig) (neo4j.DriverWithContext, error) {
	// Note: For TLS encryption, use bolt+s:// or neo4j+s:// in the URI
	driver, err := neo4j.NewDriverWithContext(config.URI, neo4j.BasicAuth(config.Username, config.Password, ""), func(c *neo4j.Config) {
		c.MaxConnectionPoolSize = config.MaxConnPool
		c.ConnectionAcquisitionTimeout = 30 * time.Second
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return driver, nil
}

func DisconnectNeo4j(ctx context.Context, driver neo4j.DriverWithContext) error {
	return driver.Close(ctx)
}
