BINARY=enos
VERSION=$$($(CURRENT_DIRECTORY)/.release/build-scripts/version.sh version/version.go)
GIT_SHA=$$(git rev-parse HEAD)
CURRENT_DIRECTORY := $(shell pwd)
BUILD_BINARY_PATH=${CURRENT_DIRECTORY}/dist/${BINARY}
REPO=github.com/hashicorp/enos
GO_BUILD_TAGS=-tags osusergo,netgo
GO_LD_FLAGS=-ldflags="-extldflags=-static -X ${REPO}/internal/version.Version=${VERSION} -X ${REPO}/internal/version.GitSHA=${GIT_SHA}"
GO_GC_FLAGS=-gcflags="all=-N -l"
GORACE=GORACE=log_path=/tmp/enos-gorace.log
TEST_ACC=ENOS_ACC=1
TEST_ACC_EXT=ENOS_ACC=1 ENOS_EXT=1

default: build

.PHONY: generate
generate: generate-proto

.PHONY: generate-proto
generate-proto:
	pushd proto/hashicorp/enos/v1 && buf generate

.PHONY: build
build:
	CGO_ENABLED=0 go build ${GO_BUILD_TAGS} ${GO_LD_FLAGS} ${GO_GC_FLAGS} -o dist/${BINARY} ./command/enos

.PHONY: build-race
build-race:
	# We can't reliably pass in the GO_GC_FLAGS for the race detector on darwin
	# https://github.com/golang/go/issues/54291
	${GORACE} go build -race ${GO_BUILD_TAGS} ${GO_LD_FLAGS} -o dist/${BINARY} ./command/enos

.PHONY: test
test:
	${GORACE} go test -race ./... -v $(TESTARGS) -timeout=5m -parallel=4

.PHONY: test-acc
test-acc: build-race
	${TEST_ACC} ${GORACE} ENOS_BINARY_PATH=${BUILD_BINARY_PATH} go test -race ./... -v $(TESTARGS) -timeout 120m

.PHONY: test-acc-ext
test-acc-ext: build-race
	${TEST_ACC_EXT} ${GORACE} ENOS_BINARY_PATH=${BUILD_BINARY_PATH} go test -race ./... -v $(TESTARGS) -timeout 120m

.PHONY: lint
lint: lint-golang lint-proto

.PHONY: lint-golang
lint-golang:
	golangci-lint run -v

.PHONY: lint-proto
lint-proto:
	pushd proto && buf lint

.PHONY: fmt
fmt: fmt-golang fmt-proto

.PHONY: fmt-check
fmt-check: fmt-check-golang fmt-check-proto

.PHONY: fmt-golang
fmt-golang:
	gofumpt -w -l .

.PHONY: fmt-check-golang
fmt-check-golang:
	gofumpt -d -l .

.PHONY: fmt-proto
fmt-proto:
	buf format proto -w

.PHONY: fmt-check-proto
fmt-check-proto:
	buf format proto -d --exit-code

.PHONY: clean
clean:
	rm -rf dist enos

.PHONY: deps
deps: deps-build deps-lint

.PHONY: deps-build
deps-build:
	go install github.com/golang/protobuf/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

.PHONY: deps-lint
deps-lint:
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: version
version:
	@$(CURRENT_DIRECTORY)/.release/build-scripts/version.sh version/version.go
