package csg

import "context"

//nolint:interfacebloat // it's fine here — we need this single large interface for use
type errorFormatterService interface {
	// ErrorNoWrap wraps an error without additional formatting (used to avoid linter warnings).
	ErrorNoWrap(err error) error
	// ErrNoWrap is the same as ErrorNoWrap — just a shorter alias.
	ErrNoWrap(err error) error
	ErrorOnly(err error, details ...string) error
	Error(err error, details ...string) error
	Errorf(err error, format string, args ...interface{}) error
	NewError(details ...string) error
	NewErrorf(format string, args ...interface{}) error
}

// сsgHealthcheckService defines the methods for checking CSG-service health.
type сsgHealthcheckService interface {
	IsHealed(ctx context.Context) error
	IsReadyForStartUp(ctx context.Context) error
	IsReadyForWork(ctx context.Context) error
}

// сsgRunnerService defines the lifecycle methods of the service.
type сsgRunnerService interface {
	Init(ctx context.Context) error
	Run(ctx context.Context) error
}

type baseConfigService interface {
	IsProd() bool
	IsStage() bool
	IsTest() bool
	IsDev() bool
	IsDebug() bool
	IsLocal() bool
}
