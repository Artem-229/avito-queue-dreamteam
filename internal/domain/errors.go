package domain

import "errors"

var (
	ErrUserAlreadyInQueue = errors.New("user is already in the queue")
)
