package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/orkestra/backend/internal/company/config"
	"github.com/orkestra/backend/internal/company/models"
)

// Client errors
var (
	ErrCompanyNotFound     = errors.New("company not found")
	ErrAPIRequestFailed    = errors.New("company API request failed")
	ErrCircuitBreakerOpen  = errors.New("circuit breaker is open")
	ErrInvalidTaxCode      = errors.New("invalid tax code")
)

// RedisClient defines the interface for Redis operations
type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
}

// CompanyAPIClient defines the interface for OpenAPI Company API operations
type CompanyAPIClient interface {
	LookupByTaxCode(ctx context.Context, taxCode string) (*models.OpenAPIBaseResponse, error)
	Ping(ctx context.Context) error
}

type companyAPIClient struct {
	config  *config.CompanyAPIConfig
	client  *http.Client
	logger  *slog.Logger
	redis   RedisClient
	breaker *circuitBreaker
}

// NewCompanyAPIClient creates a new CompanyAPIClient without Redis caching
func NewCompanyAPIClient(cfg *config.CompanyAPIConfig, logger *slog.Logger) CompanyAPIClient {
	return &companyAPIClient{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger:  logger,
		breaker: newCircuitBreaker(5, 30*time.Second),
	}
}

// NewCompanyAPIClientWithCache creates a new CompanyAPIClient with Redis caching
func NewCompanyAPIClientWithCache(cfg *config.CompanyAPIConfig, logger *slog.Logger, redis RedisClient) CompanyAPIClient {
	return &companyAPIClient{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger:  logger,
		redis:   redis,
		breaker: newCircuitBreaker(5, 30*time.Second),
	}
}

// LookupByTaxCode calls the OpenAPI Company API to look up a company by tax code
func (c *companyAPIClient) LookupByTaxCode(ctx context.Context, taxCode string) (*models.OpenAPIBaseResponse, error) {
	if taxCode == "" {
		return nil, ErrInvalidTaxCode
	}

	// Check Redis cache first
	if c.redis != nil {
		cacheKey := fmt.Sprintf("company:lookup:%s", taxCode)
		cached, err := c.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var response models.OpenAPIBaseResponse
			if err := json.Unmarshal([]byte(cached), &response); err == nil {
				c.logger.Debug("company lookup cache hit",
					"taxCode", taxCode,
				)
				return &response, nil
			}
		}
	}

	// Call external API
	path := fmt.Sprintf("/IT-start/%s", taxCode)
	respBody, statusCode, err := c.doRequestWithRetry(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	// Parse response
	var response models.OpenAPIBaseResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse company API response: %w", err)
	}

	// Handle API-level errors
	if statusCode == http.StatusNotFound || !response.Success {
		return nil, ErrCompanyNotFound
	}

	// Cache successful response in Redis
	if c.redis != nil {
		cacheKey := fmt.Sprintf("company:lookup:%s", taxCode)
		cacheData, _ := json.Marshal(response)
		if err := c.redis.Set(ctx, cacheKey, string(cacheData), c.config.CacheTTL); err != nil {
			c.logger.Warn("failed to cache company lookup",
				"taxCode", taxCode,
				"error", err,
			)
		}
	}

	return &response, nil
}

// Ping checks connectivity to the Company API
func (c *companyAPIClient) Ping(ctx context.Context) error {
	// Use a known tax code to verify API connectivity
	_, statusCode, err := c.doRequest(ctx, http.MethodGet, "/IT-start/12485671007", nil)
	if err != nil {
		return fmt.Errorf("company API ping failed: %w", err)
	}
	if statusCode >= 500 {
		return fmt.Errorf("company API returned status %d", statusCode)
	}
	return nil
}

// doRequest performs a single HTTP request to the Company API
func (c *companyAPIClient) doRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, int, error) {
	if !c.breaker.Allow() {
		return nil, 0, ErrCircuitBreakerOpen
	}

	url := c.config.GetEndpoint(path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.breaker.RecordFailure()
		return nil, 0, fmt.Errorf("%w: %v", ErrAPIRequestFailed, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.breaker.RecordFailure()
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 500 {
		c.breaker.RecordFailure()
		return respBody, resp.StatusCode, fmt.Errorf("%w: status %d", ErrAPIRequestFailed, resp.StatusCode)
	}

	c.breaker.RecordSuccess()
	return respBody, resp.StatusCode, nil
}

// doRequestWithRetry performs an HTTP request with exponential backoff retry
func (c *companyAPIClient) doRequestWithRetry(ctx context.Context, method, path string, body io.Reader) ([]byte, int, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			c.logger.Debug("retrying company API request",
				"attempt", attempt,
				"backoff", backoff,
				"path", path,
			)

			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
			}
		}

		respBody, statusCode, err := c.doRequest(ctx, method, path, body)
		if err == nil {
			return respBody, statusCode, nil
		}

		// Don't retry on 4xx client errors
		if statusCode >= 400 && statusCode < 500 {
			return respBody, statusCode, nil
		}

		lastErr = err
	}

	return nil, 0, fmt.Errorf("all retry attempts exhausted: %w", lastErr)
}

// circuitBreaker implements a simple circuit breaker pattern
type circuitBreaker struct {
	mu              sync.Mutex
	failures        int
	maxFailures     int
	resetTimeout    time.Duration
	lastFailureTime time.Time
	state           string // closed, open, half-open
}

func newCircuitBreaker(maxFailures int, resetTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        "closed",
	}
}

func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case "closed":
		return true
	case "open":
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = "half-open"
			return true
		}
		return false
	case "half-open":
		return true
	}
	return false
}

func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = "closed"
}

func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = "open"
	}
}
