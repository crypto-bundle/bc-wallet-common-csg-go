/* Package csg provides access to the Code Style Guide (CSG) of the crypto-bundle project
 * through an embedded filesystem (Go Embed).
 *
 * CSG is a collection of rules and recommendations for developing Go applications within
 * the GreenPayment / crypto-bundle project. It includes conventions for architecture,
 * error handling, configuration, PostgreSQL, NATS, HTTP, gRPC usage,
 * as well as Git-flow rules, tooling settings, and repository structures.
 *
 * # Purpose
 *
 * The csg package ensures that every developer always has the latest
 * version of the Code Style Guide alongside the application binary built
 * using this library. This is achieved through the go:embed mechanism, which incorporates
 * all documentation and instruction files
 * directly into the compiled bytecode.
 *
 * # Document Structure
 *
 * The embedded data includes the following CSG sections:
 *   - Environment Setup (setup) — Git configuration
 *   - Git Flow — branching rules and PR workflow
 *   - ToolKit — set of project libraries and utilities
 *   - Repository Layouts — directory and file structure
 *   - App Development — Go coding rules (architecture, errors, config, DB, HTTP, gRPC, NATS, Processing)
 *   - Processes — scope boundaries
 *   - Diagrams — integration diagrams
 *   - Examples — licensing and configuration file examples
 *
 * # How to Read CSG
 *
 * This package ships with complete documentation embedded into the binary via Go Embed.
 *
 * AGENTS.md in the same directory contains a condensed summary of the main rules
 * and recommendations for quick reference. The full text of each document
 * is accessible through the EFS at path `pkg/csg/docs/`.
 *
 * For programmatic work with the documentation, use the methods exported as Public API:
 * ListDocFiles(prefix) — get the list of all documents with an optional filter
 * FileContent(path)    — read the contents of any file
 * WalkDoc(fn)          — traverse all documents recursively
 *
 * Each section in pkg/csg/docs/ describes a separate aspect of development in the project.
 * Start reading from overview.md or from the app-development/overview.md section.
 */

package csg
