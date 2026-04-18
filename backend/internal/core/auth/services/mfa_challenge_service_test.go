package services

import (
	"context"
	"testing"
)

func TestMFAChallengeBeginAndConsume(t *testing.T) {
	store := NewMemoryOAuthStateStore()
	svc := NewMFAChallengeService(store)

	ch, err := svc.Begin(context.Background(), "u-1", MFAPurposeEnroll, "SECRETBASE32")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if ch.ID == "" || ch.UserUUID != "u-1" || ch.PendingSecret != "SECRETBASE32" {
		t.Fatalf("unexpected challenge payload: %+v", ch)
	}

	got, err := svc.Consume(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got.UserUUID != "u-1" {
		t.Fatalf("consumed wrong challenge: %+v", got)
	}

	// Second consume fails because the first one deleted the record.
	if _, err := svc.Consume(context.Background(), ch.ID); err != ErrMFAChallengeNotFound {
		t.Fatalf("expected ErrMFAChallengeNotFound, got %v", err)
	}
}

func TestMFAChallengeIncrementAttemptsCapsOut(t *testing.T) {
	store := NewMemoryOAuthStateStore()
	svc := NewMFAChallengeService(store)

	ch, err := svc.Begin(context.Background(), "u-2", MFAPurposeLogin, "")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for i := 1; i <= MFAMaxAttempts; i++ {
		if _, err := svc.IncrementAttempts(context.Background(), ch.ID); err != nil {
			t.Fatalf("increment %d: %v", i, err)
		}
	}
	// The MFAMaxAttempts-th increment should have deleted the challenge.
	if _, err := svc.Peek(context.Background(), ch.ID); err != ErrMFAChallengeNotFound {
		t.Fatalf("challenge not deleted after cap: %v", err)
	}
}

func TestMFAChallengeRequiresUserUUID(t *testing.T) {
	svc := NewMFAChallengeService(NewMemoryOAuthStateStore())
	if _, err := svc.Begin(context.Background(), "", MFAPurposeEnroll, ""); err == nil {
		t.Fatalf("expected error for empty userUUID")
	}
}
