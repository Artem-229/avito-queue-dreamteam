package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// TestStatusPresentation_HasMessageAndNextStep — INV-8: "у каждого состояния,
// отдаваемого фронту, есть message и next_step. Состояние без них — «серая
// зона», запрещённая кейсом." Табличный тест над statusPresentation
// (services/queue.go), без сети — гоняет по всем статусам очереди, которые
// реально могут попасть в domain.QueueStatusResponse.
func TestStatusPresentation_HasMessageAndNextStep(t *testing.T) {
	tests := []struct {
		name   string
		status domain.QueueStatus
	}{
		{"waiting", domain.QueueStatusWaiting},
		{"granted", domain.QueueStatusGranted},
		{"purchased", domain.QueueStatusPurchased},
		{"expired", domain.QueueStatusExpired},
		{"sold_out", domain.QueueStatusSoldOut},
		{"cancelled", domain.QueueStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presentation, ok := statusPresentation[tt.status]
			require.True(t, ok, "status %q must have a presentation entry, otherwise it is a grey zone forbidden by INV-8", tt.status)
			require.NotEmpty(t, presentation.Message, "message must not be empty for status %q", tt.status)
			require.NotEmpty(t, presentation.NextStep.Kind, "next_step.kind must not be empty for status %q", tt.status)
			require.NotEmpty(t, presentation.NextStep.Label, "next_step.label must not be empty for status %q", tt.status)
		})
	}
}
