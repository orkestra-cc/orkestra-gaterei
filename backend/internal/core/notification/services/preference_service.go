package services

import (
	"context"

	"github.com/orkestra/backend/internal/core/notification/models"
	"github.com/orkestra/backend/internal/core/notification/repository"
)

// PreferenceService encapsulates opt-in/out logic and the decision of whether
// a given category can be delivered for a given recipient.
type PreferenceService interface {
	CanDeliver(ctx context.Context, userUUID, category, channel, notificationType string) (bool, error)
	List(ctx context.Context, userUUID string) ([]*models.PreferenceDoc, error)
	Set(ctx context.Context, userUUID, category, channel string, optedIn bool) error
}

type preferenceService struct {
	repo repository.PreferenceRepository
}

func NewPreferenceService(repo repository.PreferenceRepository) PreferenceService {
	return &preferenceService{repo: repo}
}

func (s *preferenceService) CanDeliver(ctx context.Context, userUUID, category, channel, notificationType string) (bool, error) {
	// Transactional notifications always deliver — they're required for the
	// core product to function.
	if notificationType == models.TypeTransactional {
		return true, nil
	}
	if userUUID == "" {
		return true, nil
	}
	pref, err := s.repo.GetPreference(ctx, userUUID, category, channel)
	if err != nil {
		return false, err
	}
	if pref == nil {
		return true, nil // default: opted in until user opts out
	}
	return pref.OptedIn, nil
}

func (s *preferenceService) List(ctx context.Context, userUUID string) ([]*models.PreferenceDoc, error) {
	return s.repo.ListByUser(ctx, userUUID)
}

func (s *preferenceService) Set(ctx context.Context, userUUID, category, channel string, optedIn bool) error {
	return s.repo.UpsertPreference(ctx, &models.PreferenceDoc{
		UserUUID: userUUID,
		Category: category,
		Channel:  channel,
		OptedIn:  optedIn,
	})
}
