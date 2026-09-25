package csg

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"
)

// csgService is the main service component of the csg package.
// It encapsulates the embedded filesystem and provides
// a unified interface for working with the Code Style Guide.
type csgService struct {
	configSvc baseConfigService
	docs      embed.FS
	agents    embed.FS
	errorFmt  errorFormatterService
	logger    *slog.Logger
}

// New creates a new instance of the service with the given dependencies.
func New(baseCfgSvc baseConfigService,
	errorFmtSvc errorFormatterService,
	loggerSvc *slog.Logger,
) *csgService {
	return &csgService{
		configSvc: baseCfgSvc,
		docs:      embeddedDocs,
		agents:    embeddedAgents,
		errorFmt:  errorFmtSvc,
		logger:    loggerSvc,
	}
}

// -------------------------------------------------------------------
// Healthcheck
// -------------------------------------------------------------------

// IsHealed checks that the "docs" directory exists in EmbeddedDocs.
func (s *csgService) IsHealed(ctx context.Context) error {
	info, err := fs.Stat(s.docs, "docs")
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return nil
}

// IsReady returns nil — the service is fully ready for use.
func (s *csgService) IsReady(ctx context.Context) error {
	return s.IsHealed(ctx)
}

// -------------------------------------------------------------------
// Lifecycle
// -------------------------------------------------------------------

// Init initializes the service. Currently no additional steps are required.
func (s *csgService) Init(ctx context.Context) error {
	return nil
}

// Run starts the main logic of the CSG-service.
func (s *csgService) Run(ctx context.Context) error {
	return s.PrintUsage(os.Stdout)
}
