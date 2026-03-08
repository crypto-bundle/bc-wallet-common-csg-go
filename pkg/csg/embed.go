package csg

import (
	"embed"
	"io/fs"
)

// EmbeddedDocs содержит всю документацию Code Style Guide проекта crypto-bundle.
// Структура папок внутри сохраняется — пути доступа начинаются с "docs/".
//
//go:embed docs/app-development/architecture/file_structure.md
//go:embed docs/app-development/architecture/main_go.md
//go:embed docs/app-development/architecture/package_structure.md
//go:embed docs/app-development/config/index.md
//go:embed docs/app-development/errors/codes.md
//go:embed docs/app-development/errors/handling.md
//go:embed docs/app-development/grpc/index.md
//go:embed docs/app-development/http/index.md
//go:embed docs/app-development/nats/index.md
//go:embed docs/app-development/overview.md
//go:embed docs/app-development/postgresql/data_layer.md
//go:embed docs/app-development/postgresql/delete.md
//go:embed docs/app-development/postgresql/index.md
//go:embed docs/app-development/postgresql/insert.md
//go:embed docs/app-development/postgresql/select.md
//go:embed docs/app-development/postgresql/transactions.md
//go:embed docs/app-development/postgresql/update.md
//go:embed docs/app-development/processing/index.md
//go:embed docs/diagrams/integration-branch.puml
//go:embed docs/examples/env/env-hdwallet-controller-manager-api-example.env
//go:embed docs/examples/license/LICENSE.txt
//go:embed docs/examples/license/README.md
//go:embed docs/examples/license/cmd.go
//go:embed docs/git-flow/git_flow.md
//go:embed docs/processes/perimeter_rules.md
//go:embed docs/repository-layouts/app/code.md
//go:embed docs/repository-layouts/app/deployment.md
//go:embed docs/repository-layouts/builders.md
//go:embed docs/repository-layouts/cicd.md
//go:embed docs/repository-layouts/configuration-files.md
//go:embed docs/repository-layouts/documentation.md
//go:embed docs/repository-layouts/license.md
//go:embed docs/repository-layouts/linters.md
//go:embed docs/repository-layouts/migrations/index.md
//go:embed docs/repository-layouts/overview.md
//go:embed docs/review/github_code_review.md
//go:embed docs/setup/git_setup.md
//go:embed docs/toolkit/copywrite.md
//go:embed docs/toolkit/golangci-lint.md
//go:embed docs/toolkit/migrator.md
//go:embed docs/toolkit/overview.md
//go:embed AGENTS.md
var EmbeddedDocs embed.FS

// EmbeddedAgents содержит инструкции для AI-агентов из AGENTS.md.
//
//go:embed AGENTS.md
var EmbeddedAgents embed.FS

// WalkDoc recursively walks all embedded documentation files
// and calls the provided function for each entry.
func WalkDoc(fn func(path string, d fs.DirEntry, err error) error) error {
	return fs.WalkDir(EmbeddedDocs, "docs", fn)
}
