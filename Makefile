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
	cd tools && go install honnef.co/go/tools/cmd/staticcheck golang.org/x/vuln/cmd/govulncheck github.com/rhysd/actionlint/cmd/actionlint
release:
	DEVDOCK_DIST_DIR="$(DEVDOCK_DIST_DIR)" bash scripts/build-release.sh
clean-release:
	python3 scripts/release_output.py "$(DEVDOCK_DIST_DIR)"
source-archive:
	bash scripts/package-source.sh "$(SOURCE_ARCHIVE)"
