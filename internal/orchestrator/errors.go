package orchestrator

import "errors"

var (
	ErrPlannerNotInitialized      = errors.New("planner not initialized")
	ErrUnsupportedTransitionShape = errors.New("unsupported transition shape")
	ErrInvalidConclusion          = errors.New("invalid conclusion")
	ErrDispatchConflict           = errors.New("dispatch conflict")
)
