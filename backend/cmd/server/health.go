package main

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// registerHealthEndpoints registers /health and /ready endpoints.
func registerHealthEndpoints(api huma.API, db *mongo.Database, redisClient *redis.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      "GET",
		Path:        "/health",
		Summary:     "Health check",
		Description: "Returns the health status of the application",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Status  string            `json:"status"`
			Time    string            `json:"time"`
			Version string            `json:"version"`
			Checks  map[string]string `json:"checks"`
		}
	}, error) {
		checks := map[string]string{}

		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := db.Client().Ping(ctx, nil); err != nil {
			checks["mongodb"] = "down"
		} else {
			checks["mongodb"] = "up"
		}

		if err := redisClient.Ping(ctx).Err(); err != nil {
			checks["redis"] = "down"
		} else {
			checks["redis"] = "up"
		}

		status := "healthy"
		for _, check := range checks {
			if check == "down" {
				status = "unhealthy"
				break
			}
		}

		return &struct {
			Body struct {
				Status  string            `json:"status"`
				Time    string            `json:"time"`
				Version string            `json:"version"`
				Checks  map[string]string `json:"checks"`
			}
		}{
			Body: struct {
				Status  string            `json:"status"`
				Time    string            `json:"time"`
				Version string            `json:"version"`
				Checks  map[string]string `json:"checks"`
			}{
				Status:  status,
				Time:    time.Now().UTC().Format(time.RFC3339),
				Version: "1.0.0",
				Checks:  checks,
			},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "readiness-check",
		Method:      "GET",
		Path:        "/ready",
		Summary:     "Readiness check",
		Description: "Returns whether the application is ready to accept requests",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Ready bool `json:"ready"`
		}
	}, error) {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		ready := true
		if err := db.Client().Ping(ctx, nil); err != nil {
			ready = false
		}
		if err := redisClient.Ping(ctx).Err(); err != nil {
			ready = false
		}

		return &struct {
			Body struct {
				Ready bool `json:"ready"`
			}
		}{
			Body: struct {
				Ready bool `json:"ready"`
			}{
				Ready: ready,
			},
		}, nil
	})
}
