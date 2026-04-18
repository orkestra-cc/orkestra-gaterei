package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MFAChallengePurpose categorises the flow a challenge belongs to so the
// wrong purpose can't be consumed where it shouldn't (e.g. an enroll
// challenge satisfying a login step).
type MFAChallengePurpose string

const (
	MFAPurposeEnroll MFAChallengePurpose = "enroll"
	MFAPurposeLogin  MFAChallengePurpose = "login"   // consumed in Block B
	MFAPurposeStepUp MFAChallengePurpose = "step_up" // consumed in Block D
)

// MFAChallengeTTL is the Redis TTL applied to newly issued challenges.
// Five minutes leaves room for the user to open an authenticator app and
// type a code without being so long that a leaked challengeId stays useful.
const MFAChallengeTTL = 5 * time.Minute

// MFAMaxAttempts caps the number of guesses per challenge. Exceeding this
// causes the challenge to be deleted — the client must start a fresh flow.
const MFAMaxAttempts = 5

// ErrMFAChallengeNotFound is returned when Get/Consume/IncrementAttempts is
// called with a challenge ID that does not exist (expired, consumed, or
// exhausted). Callers treat it as "unauthenticated".
var ErrMFAChallengeNotFound = errors.New("mfa challenge not found")

// MFAChallenge is the Redis-stored payload. PendingSecret is only populated
// for enrollment challenges — login and step-up flows reference the user's
// already-persisted factor instead.
type MFAChallenge struct {
	ID            string              `json:"id"`
	UserUUID      string              `json:"userUuid"`
	Purpose       MFAChallengePurpose `json:"purpose"`
	PendingSecret string              `json:"pendingSecret,omitempty"`
	Attempts      int                 `json:"attempts"`
	CreatedAt     time.Time           `json:"createdAt"`
	ExpiresAt     time.Time           `json:"expiresAt"`
}

// MFAChallengeService issues, looks up, and consumes short-lived challenges
// in Redis. Kept small on purpose: the TOTP verification itself lives in
// the MFA service; this layer is just secure state for a single flow.
type MFAChallengeService interface {
	Begin(ctx context.Context, userUUID string, purpose MFAChallengePurpose, pendingSecret string) (*MFAChallenge, error)
	Peek(ctx context.Context, id string) (*MFAChallenge, error)
	Consume(ctx context.Context, id string) (*MFAChallenge, error)
	IncrementAttempts(ctx context.Context, id string) (int, error)
}

type mfaChallengeService struct {
	store OAuthStateStore // reused: same Set/Get/Delete surface we already have
}

// NewMFAChallengeService constructs the service on top of any storage that
// satisfies OAuthStateStore — notably RedisOAuthStateStore. Sharing the
// store type avoids a second Redis adapter for a nearly-identical pattern.
func NewMFAChallengeService(store OAuthStateStore) MFAChallengeService {
	return &mfaChallengeService{store: store}
}

func (s *mfaChallengeService) Begin(ctx context.Context, userUUID string, purpose MFAChallengePurpose, pendingSecret string) (*MFAChallenge, error) {
	if userUUID == "" {
		return nil, fmt.Errorf("userUUID is required")
	}
	now := time.Now()
	ch := &MFAChallenge{
		ID:            uuid.NewString(),
		UserUUID:      userUUID,
		Purpose:       purpose,
		PendingSecret: pendingSecret,
		Attempts:      0,
		CreatedAt:     now,
		ExpiresAt:     now.Add(MFAChallengeTTL),
	}

	payload, err := json.Marshal(ch)
	if err != nil {
		return nil, fmt.Errorf("marshal mfa challenge: %w", err)
	}
	if err := s.store.Set(ctx, buildMFAChallengeKey(ch.ID), payload, MFAChallengeTTL); err != nil {
		return nil, fmt.Errorf("store mfa challenge: %w", err)
	}
	return ch, nil
}

func (s *mfaChallengeService) Peek(ctx context.Context, id string) (*MFAChallenge, error) {
	if id == "" {
		return nil, ErrMFAChallengeNotFound
	}
	raw, err := s.store.Get(ctx, buildMFAChallengeKey(id))
	if err != nil {
		return nil, ErrMFAChallengeNotFound
	}
	var ch MFAChallenge
	if err := json.Unmarshal(raw, &ch); err != nil {
		return nil, fmt.Errorf("unmarshal mfa challenge: %w", err)
	}
	if time.Now().After(ch.ExpiresAt) {
		_ = s.store.Delete(ctx, buildMFAChallengeKey(id))
		return nil, ErrMFAChallengeNotFound
	}
	return &ch, nil
}

func (s *mfaChallengeService) Consume(ctx context.Context, id string) (*MFAChallenge, error) {
	ch, err := s.Peek(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = s.store.Delete(ctx, buildMFAChallengeKey(id))
	return ch, nil
}

// IncrementAttempts bumps the counter and returns the new value. When the
// counter reaches MFAMaxAttempts the challenge is deleted, forcing the
// client to start over — cheapest rate-limiter for a 6-digit code.
func (s *mfaChallengeService) IncrementAttempts(ctx context.Context, id string) (int, error) {
	ch, err := s.Peek(ctx, id)
	if err != nil {
		return 0, err
	}
	ch.Attempts++
	if ch.Attempts >= MFAMaxAttempts {
		_ = s.store.Delete(ctx, buildMFAChallengeKey(id))
		return ch.Attempts, nil
	}
	remaining := time.Until(ch.ExpiresAt)
	if remaining <= 0 {
		_ = s.store.Delete(ctx, buildMFAChallengeKey(id))
		return ch.Attempts, ErrMFAChallengeNotFound
	}
	payload, err := json.Marshal(ch)
	if err != nil {
		return ch.Attempts, fmt.Errorf("marshal mfa challenge: %w", err)
	}
	if err := s.store.Set(ctx, buildMFAChallengeKey(id), payload, remaining); err != nil {
		return ch.Attempts, fmt.Errorf("store mfa challenge: %w", err)
	}
	return ch.Attempts, nil
}

func buildMFAChallengeKey(id string) string {
	return fmt.Sprintf("mfa:challenge:%s", id)
}
