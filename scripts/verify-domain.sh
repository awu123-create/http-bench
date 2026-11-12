#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
go vet ./...
go test -race ./... -count=1
go build ./...
