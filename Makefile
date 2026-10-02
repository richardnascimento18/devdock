.PHONY: build test race vet fmt check tools release clean-release source-archive

DEVDOCK_DIST_DIR ?= dist
SOURCE_ARCHIVE ?= /tmp/devdock-source.zip

build:
	go build -o devdock .
test:
	go test ./...
race:
	go test -race ./...
vet:
	go vet ./...
fmt:
	gofmt -w .
check:
	bash scripts/check.sh
tools:
	# go install honors the caller's GOBIN; check.sh uses it on PATH.
	go install honnef.co/go/tools/cmd/staticcheck@v0.8.1
	go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
	go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
release:
	DEVDOCK_DIST_DIR="$(DEVDOCK_DIST_DIR)" bash scripts/build-release.sh
clean-release:
	python3 scripts/release_output.py "$(DEVDOCK_DIST_DIR)"
source-archive:
	bash scripts/package-source.sh "$(SOURCE_ARCHIVE)"
