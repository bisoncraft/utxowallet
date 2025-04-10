#!/usr/bin/env bash
set -ex

GV=$(go version | sed "s/^.*go\([0-9.]*\).*/\1/")
echo "Go version: $GV"

# Run `go mod tidy` and fail if the git status of go.mod and/or
# go.sum changes. Only do this for the latest Go version.
if [[ "$GV" =~ ^1.23 ]]; then
	MOD_STATUS=$(git status --porcelain go.mod go.sum)
	go mod tidy
	UPDATED_MOD_STATUS=$(git status --porcelain go.mod go.sum)
	if [ "$UPDATED_MOD_STATUS" != "$MOD_STATUS" ]; then
		echo "$m: running 'go mod tidy' modified go.mod and/or go.sum"
	git diff --unified=0 go.mod go.sum
		exit 1
	fi
fi

# run tests
env GORACE="halt_on_error=1" go test -race -short ./...

# check linters
golangci-lint -c ./.golangci.yml run
