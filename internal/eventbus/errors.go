package eventbus

import "errors"

var (
	ErrBusNotInitialized = errors.New("bus not initialized")
	ErrEventChainBroken  = errors.New("event chain broken")
	ErrLeaseConflict     = errors.New("lease conflict")
	ErrLeaseExpired      = errors.New("lease expired")
	ErrOffsetRegression  = errors.New("offset regression")
	ErrInterruptNotFound = errors.New("interrupt not found")
	ErrInvalidEventType  = errors.New("invalid event type")
)
