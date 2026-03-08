package csg

import (
	"fmt"
	"io/fs"
)

// RequiredPaths определяет список обязательных документов CSG,
// которые должны присутствовать во встроенной файловой системе.
var RequiredPaths = []string{
	"docs/app-development/overview.md",
	"docs/app-development/architecture/package_structure.md",
	"docs/app-development/config/index.md",
	"docs/app-development/errors/handling.md",
	"docs/app-development/postgresql/index.md",
	"docs/toolkit/overview.md",
	"docs/repository-layouts/overview.md",
	"docs/setup/git_setup.md",
	"docs/git-flow/git_flow.md",
	"AGENTS.md",
}

// Validate проверяет наличие всех обязательных файлов
// во встроенной файловой системе EmbeddedDocs и возвращает ошибку,
// если какой-либо документ отсутствует.
func Validate() error {
	var missing []string

	for _, path := range RequiredPaths {
		info, err := fs.Stat(EmbeddedDocs, path)
		if err != nil || info.IsDir() {
			missing = append(missing, path)
			continue
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("csg: %d required document(s) missing from embedded filesystem: %v",
			len(missing), missing)
	}

	return nil
}
