//go:build teststub

package backend

// This file intentionally contains no SQLite driver. It exists only so
// `go test -tags teststub ./...` can type-check the project in an offline
// environment. Do not use the teststub tag for a real build.
