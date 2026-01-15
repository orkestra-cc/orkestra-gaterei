package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/orkestra/backend/internal/billing/models"
	"github.com/orkestra/backend/internal/billing/repository"
	"github.com/orkestra/backend/internal/billing/services"
)

// PollingJob handles periodic polling of SDI notifications
type PollingJob struct {
	openAPIClient    services.OpenAPIClient
	invoiceRepo      repository.InvoiceRepository
	notificationRepo repository.NotificationRepository
	logger           *slog.Logger
	interval         time.Duration
	stopChan         chan struct{}
	running          bool
}

// NewPollingJob creates a new polling job
func NewPollingJob(
	openAPIClient services.OpenAPIClient,
	invoiceRepo repository.InvoiceRepository,
	notificationRepo repository.NotificationRepository,
	logger *slog.Logger,
	interval time.Duration,
) *PollingJob {
	return &PollingJob{
		openAPIClient:    openAPIClient,
		invoiceRepo:      invoiceRepo,
		notificationRepo: notificationRepo,
		logger:           logger,
		interval:         interval,
		stopChan:         make(chan struct{}),
	}
}

// Start begins the polling job
func (j *PollingJob) Start(ctx context.Context) {
	if j.running {
		j.logger.Warn("polling job already running")
		return
	}

	j.running = true
	j.logger.Info("starting SDI notification polling job",
		"interval", j.interval.String(),
	)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	// Run immediately on start
	j.poll(ctx)

	for {
		select {
		case <-ticker.C:
			j.poll(ctx)
		case <-j.stopChan:
			j.logger.Info("stopping SDI notification polling job")
			j.running = false
			return
		case <-ctx.Done():
			j.logger.Info("SDI notification polling job stopped due to context cancellation")
			j.running = false
			return
		}
	}
}

// Stop stops the polling job
func (j *PollingJob) Stop() {
	if j.running {
		close(j.stopChan)
	}
}

// Poll manually triggers a poll (useful for API endpoint)
func (j *PollingJob) Poll(ctx context.Context) error {
	return j.poll(ctx)
}

func (j *PollingJob) poll(ctx context.Context) error {
	j.logger.Debug("polling for SDI notifications")

	// Get last polling state
	state, err := j.notificationRepo.GetPollingState(ctx)
	if err != nil {
		j.logger.Error("failed to get polling state", "error", err)
		return err
	}

	// Fetch notifications from OpenAPI
	fromDate := state.LastPolledAt.Add(-1 * time.Hour) // Overlap by 1 hour to avoid missing notifications
	notifications, err := j.openAPIClient.GetNotifications(ctx, fromDate)
	if err != nil {
		j.logger.Error("failed to fetch notifications from OpenAPI", "error", err)

		// Update state with error
		state.LastError = err.Error()
		now := time.Now()
		state.LastErrorAt = &now
		state.ConsecutiveErrors++
		_ = j.notificationRepo.UpdatePollingState(ctx, state)

		return err
	}

	j.logger.Info("fetched notifications from OpenAPI",
		"count", len(notifications),
		"fromDate", fromDate.Format(time.RFC3339),
	)

	// Process each notification
	var lastNotificationTime time.Time
	processedCount := 0

	for _, n := range notifications {
		if err := j.processNotification(ctx, &n); err != nil {
			j.logger.Error("failed to process notification",
				"uuid", n.UUID,
				"type", n.Type,
				"error", err,
			)
			continue
		}
		processedCount++

		if n.Date.After(lastNotificationTime) {
			lastNotificationTime = n.Date
		}
	}

	// Update polling state
	state.LastPolledAt = time.Now()
	if !lastNotificationTime.IsZero() {
		state.LastNotificationAt = &lastNotificationTime
	}
	state.TotalPolled += int64(processedCount)
	state.LastError = ""
	state.LastErrorAt = nil
	state.ConsecutiveErrors = 0

	if err := j.notificationRepo.UpdatePollingState(ctx, state); err != nil {
		j.logger.Error("failed to update polling state", "error", err)
	}

	j.logger.Info("polling completed",
		"processedCount", processedCount,
		"totalNotifications", len(notifications),
	)

	return nil
}

func (j *PollingJob) processNotification(ctx context.Context, n *services.OpenAPINotification) error {
	// Find the associated invoice
	invoice, err := j.invoiceRepo.GetByOpenAPIUUID(ctx, n.InvoiceUUID)
	if err != nil {
		// Invoice not found in our system - might be from a different source
		j.logger.Warn("invoice not found for notification",
			"notificationUUID", n.UUID,
			"invoiceUUID", n.InvoiceUUID,
		)
		// Still save the notification for audit purposes
		return j.saveNotification(ctx, n, "")
	}

	// Update invoice status based on notification type
	newStatus := invoice.Status
	newSDIStatus := models.SDIStatus(n.Type)

	switch models.NotificationType(n.Type) {
	case models.NotificationRC: // Ricevuta di Consegna
		newStatus = models.StatusDelivered
		j.logger.Info("invoice delivered",
			"invoiceUUID", invoice.UUID,
			"number", invoice.Number,
		)

	case models.NotificationNS: // Notifica di Scarto
		newStatus = models.StatusRejected
		j.logger.Warn("invoice rejected by SDI",
			"invoiceUUID", invoice.UUID,
			"number", invoice.Number,
			"errorCode", n.ErrorCode,
			"errorDescription", n.ErrorDescription,
		)

	case models.NotificationMC: // Mancata Consegna
		// Status remains Sent, but SDI status changes
		j.logger.Warn("invoice delivery failed",
			"invoiceUUID", invoice.UUID,
			"number", invoice.Number,
		)

	case models.NotificationNE: // Notifica Esito (PA only)
		if n.Outcome == string(models.OutcomeAccepted) {
			newStatus = models.StatusAccepted
			j.logger.Info("invoice accepted by PA",
				"invoiceUUID", invoice.UUID,
				"number", invoice.Number,
			)
		} else {
			newStatus = models.StatusRejected
			j.logger.Warn("invoice rejected by PA",
				"invoiceUUID", invoice.UUID,
				"number", invoice.Number,
				"outcome", n.Outcome,
			)
		}

	case models.NotificationDT: // Decorrenza Termini
		newStatus = models.StatusAccepted // Silenzio-assenso
		j.logger.Info("invoice accepted by silence (decorrenza termini)",
			"invoiceUUID", invoice.UUID,
			"number", invoice.Number,
		)

	case models.NotificationAT: // Attestazione
		j.logger.Info("invoice transmission attested",
			"invoiceUUID", invoice.UUID,
			"number", invoice.Number,
		)
	}

	// Update invoice status
	if err := j.invoiceRepo.UpdateStatus(ctx, invoice.UUID, newStatus, newSDIStatus); err != nil {
		j.logger.Error("failed to update invoice status",
			"invoiceUUID", invoice.UUID,
			"error", err,
		)
		return err
	}

	// Save the notification
	return j.saveNotification(ctx, n, invoice.UUID)
}

func (j *PollingJob) saveNotification(ctx context.Context, n *services.OpenAPINotification, invoiceUUID string) error {
	notification := &models.SDINotification{
		UUID:             uuid.New().String(),
		InvoiceUUID:      invoiceUUID,
		OpenAPIUUID:      n.UUID,
		NotificationType: models.NotificationType(n.Type),
		NotificationDate: n.Date,
		RawContent:       n.RawContent,
		ErrorCode:        n.ErrorCode,
		ErrorDescription: n.ErrorDescription,
		Outcome:          models.NEOutcome(n.Outcome),
		Processed:        false,
		CreatedAt:        time.Now(),
	}

	return j.notificationRepo.Create(ctx, notification)
}
