package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// TestStatusPresentation_HasMessageAndNextStep — INV-8: "у каждого состояния,
// отдаваемого фронту, есть message и next_step. Состояние без них — «серая
// зона», запрещённая кейсом." Табличный тест над entryPresentation
// (services/queue_entries.go), без сети — гоняет по всем статусам участия,
// которые реально могут попасть в domain.QueueStatusResponse.
//
// not_in_queue в списке намеренно: отсутствие записи — тоже состояние, а не
// 404, и сообщение со следующим шагом ему нужны наравне с остальными.
func TestStatusPresentation_HasMessageAndNextStep(t *testing.T) {
	tests := []struct {
		name   string
		status domain.QueueEntryStatus
	}{
		{"not_in_queue", domain.QueueEntryStatusNotInQueue},
		{"waiting", domain.QueueEntryStatusWaiting},
		{"granted", domain.QueueEntryStatusGranted},
		{"purchased", domain.QueueEntryStatusPurchased},
		{"expired", domain.QueueEntryStatusExpired},
		{"sold_out", domain.QueueEntryStatusSoldOut},
		{"cancelled", domain.QueueEntryStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presentation, ok := entryPresentation[tt.status]
			require.True(t, ok, "status %q must have a presentation entry, otherwise it is a grey zone forbidden by INV-8", tt.status)
			require.NotEmpty(t, presentation.Message, "message must not be empty for status %q", tt.status)
			require.NotEmpty(t, presentation.NextStep.Kind, "next_step.kind must not be empty for status %q", tt.status)
			require.NotEmpty(t, presentation.NextStep.Label, "next_step.label must not be empty for status %q", tt.status)
		})
	}
}
