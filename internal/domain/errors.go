package domain

import "errors"

var (
	ErrUserAlreadyInQueue = errors.New("user is already in the queue")
	ErrNoItemFound        = errors.New("no item found")
	ErrNoPurchaseRight    = errors.New("no granted purchase right for this item")
	ErrUserNotFound       = errors.New("user not found")
	ErrItemSoldOut        = errors.New("item sold out")
	ErrNoTx               = errors.New("no transaction in context")
	ErrCannotLeaveQueue   = errors.New("cannot leave queue in current state")
	// ErrStatusConflict — переход статуса не применился: запись успели перевести
	// в другое состояние либо её не существует в ожидаемом состоянии.
	ErrStatusConflict = errors.New("status transition conflict")
)
