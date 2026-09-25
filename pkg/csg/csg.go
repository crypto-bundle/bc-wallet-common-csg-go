package csg

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

var (
	ErrMissingDocs = errors.New("missing csg docs")
)

// -------------------------------------------------------------------
// Public API (methods on csgService)
// -------------------------------------------------------------------

// Unwrap returns the entire embedded Code Style Guide documentation.
// Allows reading any files through the stdlib io/fs interface.
func (s *csgService) Unwrap() fs.FS {
	return s.docs
}

// AgentsFS returns access to the built-in AI agent instructions.
func (s *csgService) AgentsFS() fs.FS {
	return s.agents
}

// FileContent reads and returns the contents of a file from the embedded
// CSG documentation. Returns an error if the file is not found.
func (s *csgService) FileContent(path string) ([]byte, error) {
	return s.docs.ReadFile(path)
}

// ListDocFiles lists all CSG documents matching an optional
// path prefix. If prefix is empty — returns all files.
func (s *csgService) ListDocFiles(prefix string) ([]string, error) {
	var files []string
	err := s.walkDoc(func(path string, d fs.DirEntry, err error) error {
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

// PrintUsage prints a brief overview of the available CSG sections.
func (s *csgService) PrintUsage(w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintf(w, "Code Style Guide (CSG) — Built-in Documentation\n")
	fmt.Fprintf(w, "================================================\n\n")

	printedSections := map[string]bool{}
	s.walkDoc(func(path string, d fs.DirEntry, err error) error {
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
	return nil
}

// -------------------------------------------------------------------
// Internal helpers (unexported)
// -------------------------------------------------------------------

// validate checks for the presence of the root 'docs' directory in the EFS.
func (s *csgService) validate() error {
	info, err := fs.Stat(s.docs, "docs")
	if err != nil {
		return s.errorFmt.Error(err, "embedded docs FS is empty or missing")
	}

	if info.IsDir() {
		return nil
	}

	return s.errorFmt.ErrorOnly(ErrMissingDocs, "csg: expected 'docs' in embedded filesystem to be a directory")
}

// walkDoc iterates recursively over all embedded documents.
func (s *csgService) walkDoc(fn func(path string, d fs.DirEntry, err error) error) error {
	return fs.WalkDir(s.docs, "docs", fn)
}
