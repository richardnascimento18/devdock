.PHONY: build test race vet fmt check tools release

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
	bash scripts/build-release.sh
