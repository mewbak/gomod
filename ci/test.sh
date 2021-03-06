#!/usr/bin/env bash
# vim: set tabstop=4 shiftwidth=4 expandtab
set -e -u -o pipefail

PROJECT_ROOT="$(dirname "${BASH_SOURCE[0]}")/.."
cd "${PROJECT_ROOT}"

# Run all the Go tests with the race detector and generate coverage.
printf "\nRunning Go test...\n"
go test -v -race -coverprofile c.out -coverpkg=all ./...

# Run all the Bash tests.
printf "\nRunning Bash tests...\n"
./internal/completion/scripts/go
