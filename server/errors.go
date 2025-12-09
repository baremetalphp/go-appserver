package server

import "errors"

var (
	ErrWorkerDead = errors.New("worker is dead")

	ErrWorkerDraining = errors.New("worker is draining")
)
