#!/bin/sh
# All checks of the repo, used by CI and before committing.
set -e
cd "$(dirname "$0")/.."
gofmt -l . | grep . && { echo 'ERROR files above are not gofmt-formatted'; exit 1; } || true
go vet ./...
# If drawio is available, also check that the real draw.io draws wires at the computed coordinates.
if command -v drawio >/dev/null 2>&1 || command -v draw.io >/dev/null 2>&1; then
	FLOWCAST_DRAWIO_TEST=1 go test ./...
else
	echo 'drawio CLI not found: skipping render tests with the real drawio'
	go test ./...
fi
