package graph

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

// NewGraphDriver constructs the Bolt driver without dialing. The neo4j-go
// driver is lazy — it only opens TCP connections on VerifyConnectivity or
// the first session Run — so this is safe to call even when the target
// database isn't up yet. Modules that own their database's lifecycle
// (e.g. the graph module, which starts orkestra-memgraph via
// InfraContainers) should call this at Init and VerifyGraphConnection at
// Start, once the container manager has brought the server up.
func NewGraphDriver(config GraphDBConfig) (neo4j.DriverWithContext, error) {
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
	return driver, nil
}

// VerifyGraphConnection pings the database and retries for up to timeout.
// Memgraph starts accepting Bolt connections a couple of seconds before
// it's fully ready to serve queries, so callers should allow ~15–30s for
// first-boot scenarios. A per-attempt budget of 2s keeps retries snappy.
func VerifyGraphConnection(ctx context.Context, driver neo4j.DriverWithContext, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := driver.VerifyConnectivity(attemptCtx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return fmt.Errorf("failed to verify graph database connectivity within %s: %w", timeout, lastErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// NewGraphConnection builds the driver and verifies connectivity in a
// single call. Kept for backwards compatibility; prefer NewGraphDriver +
// VerifyGraphConnection when the caller manages the database lifecycle.
func NewGraphConnection(ctx context.Context, config GraphDBConfig) (neo4j.DriverWithContext, error) {
	driver, err := NewGraphDriver(config)
	if err != nil {
		return nil, err
	}
	if err := VerifyGraphConnection(ctx, driver, 5*time.Second); err != nil {
		return nil, err
	}
	return driver, nil
}

func DisconnectGraph(ctx context.Context, driver neo4j.DriverWithContext) error {
	return driver.Close(ctx)
}
