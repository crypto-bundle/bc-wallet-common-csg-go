package csg

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// Docs возвращает доступ ко всей встроенной документации Code Style Guide.
// Позволяет читать любые файлы через stdlib io/fs интерфейс.
func Docs() fs.FS {
	return EmbeddedDocs
}

// Agents возвращает доступ к встроенным инструкциям для AI-агентов.
func Agents() fs.FS {
	return EmbeddedAgents
}

// FileContent читает и возвращает содержимое файла из встроенной
// документации CSG. Возвращает ошибку, если файл не найден.
func FileContent(path string) ([]byte, error) {
	return EmbeddedDocs.ReadFile(path)
}

// ListDocFiles перечисляет все документы CSG, соответствующие опциональному
// префиксу пути. Если prefix пустой — возвращаются все файлы.
func ListDocFiles(prefix string) ([]string, error) {
	var files []string
	err := WalkDoc(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if prefix != "" && !strings.HasPrefix(path, "docs/"+prefix) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// PrintUsage выводит краткую информацию о доступных разделах CSG.
func PrintUsage(w io.Writer) {
	fmt.Fprintf(w, "Code Style Guide (CSG) — Built-in Documentation\n")
	fmt.Fprintf(w, "================================================\n\n")

	printedSections := map[string]bool{}
	WalkDoc(func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, "docs/")
		parts := strings.SplitN(rel, "/", 2)
		section := parts[0]
		if !printedSections[section] {
			fmt.Fprintf(w, "  [%s]\n", section)
			printedSections[section] = true
		}
		return nil
	})
}
