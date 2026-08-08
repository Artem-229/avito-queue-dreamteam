package domain

import "errors"

var (
	ErrUserAlreadyInQueue = errors.New("user is already in the queue")
	ErrNoItemFound        = errors.New("no item found")
	ErrNoPurchaseRight    = errors.New("no granted purchase right for this item")
	ErrUserNotFound       = errors.New("user not found")
	ErrItemSoldOut        = errors.New("item sold out")
)
