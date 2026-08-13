package domain

import (
	"time"

	"github.com/google/uuid"
)

// QueueEntryStatus — состояние участия пользователя в очереди на товар.
// После слияния queue и purchase_rights это единственный статус в системе:
// отдельного состояния права на покупку больше нет, право — это участие в
// статусе granted с непросроченным expires_at.
type QueueEntryStatus string

const (
	QueueEntryStatusWaiting   QueueEntryStatus = "waiting"
	QueueEntryStatusGranted   QueueEntryStatus = "granted"
	QueueEntryStatusPurchased QueueEntryStatus = "purchased"
	QueueEntryStatusExpired   QueueEntryStatus = "expired"
	QueueEntryStatusCancelled QueueEntryStatus = "cancelled"
	QueueEntryStatusSoldOut   QueueEntryStatus = "sold_out"

	QueueEntryStatusNotInQueue QueueEntryStatus = "not_in_queue"
)

// IsTerminal — из состояния больше нет переходов, участие завершено.
// Терминальные записи не удаляются: на них считается статистика и по ним
// видно историю попыток пользователя.
func (s QueueEntryStatus) IsTerminal() bool {
	switch s {
	case QueueEntryStatusPurchased, QueueEntryStatusExpired,
		QueueEntryStatusCancelled, QueueEntryStatusSoldOut:
		return true
	default:
		return false
	}
}

type QueueEntry struct {
	ID        uuid.UUID
	ItemID    uuid.UUID
	UserID    uuid.UUID
	Status    QueueEntryStatus
	CreatedAt time.Time
	// GrantedAt и ExpiresAt непусты ровно тогда, когда право выдавалось:
	// у сгоревшего участия они остаются для статистики.
	GrantedAt  *time.Time
	ExpiresAt  *time.Time
	ResolvedAt *time.Time
	// InitialPosition — позиция на момент входа, не меняется. Нужна, чтобы
	// показать продвижение очереди («были 9-м, сейчас 6-й»). У записей,
	// созданных до появления колонки, пуста.
	InitialPosition *int
}

// QueueSnapshot — согласованный срез состояния для конверта ответа: запись
// пользователя, его позиция, размер очереди и время сервера. Собирается одной
// выборкой вместо четырёх
type QueueSnapshot struct {
	// Entry пуст, если пользователь никогда не вставал в очередь на этот товар.
	Entry *QueueEntry
	// Position — 1 = следующий на выдачу права. Заполнена только для waiting.
	Position int
	// QueueSize — сколько человек сейчас в статусе waiting по товару.
	QueueSize    int
	ServerTime   time.Time
	TotalStock   int
	GrantedCount int
	UsedCount    int
}

// Status — состояние для конверта ответа, включая виртуальное not_in_queue.
func (s QueueSnapshot) Status() QueueEntryStatus {
	if s.Entry == nil {
		return QueueEntryStatusNotInQueue
	}
	return s.Entry.Status
}

// NextStep — подсказка для UI, что делать дальше в текущем состоянии (INV-8).
type NextStep struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

// PurchaseChance — вероятность того, что очередь дойдёт до пользователя.
// Basis говорит, на чём она посчитана: item — на конверсии этого товара,
// global — на общей по системе, default — данных ещё нет. Клиенту это нужно,
// чтобы не выдавать оценку по умолчанию за измеренную.
type PurchaseChance struct {
	Percent int    `json:"percent"`
	Basis   string `json:"basis"`
}

// QueueStatusResponse — единый конверт состояния. Все ручки очереди отвечают
// одинаково: экрану не приходится склеивать состояние из разных форматов, а у
// каждого состояния есть сообщение и следующий шаг (INV-8).
type QueueStatusResponse struct {
	Status          QueueEntryStatus `json:"status"`
	Message         string           `json:"message"`
	Position        int              `json:"position,omitempty"`
	ExpiresAt       *time.Time       `json:"expires_at,omitempty"`
	InitialPosition *int             `json:"initial_position,omitempty"`
	ProgressPercent *int             `json:"progress_percent,omitempty"`
	// QueueSize — сколько человек сейчас в статусе waiting по этому товару.
	QueueSize  int             `json:"queue_size"`
	ETASeconds *int            `json:"eta_seconds"`
	Chance     *PurchaseChance `json:"chance,omitempty"`
	// ServerTime — время Postgres, не Go (INV-6).
	ServerTime time.Time `json:"server_time"`
	NextStep   NextStep  `json:"next_step"`
	// Alternatives — похожие лоты для sold_out/expired, иначе пустой срез
	// (не nil, чтобы в JSON было [], а не null).
	Alternatives []uuid.UUID `json:"alternatives"`
}
