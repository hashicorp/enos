BINARY=enos
VERSION=$$($(CURRENT_DIRECTORY)/.release/build-scripts/version.sh version/version.go)
GIT_SHA=$$(git rev-parse HEAD)
CURRENT_DIRECTORY := $(shell pwd)
BUILD_BINARY_PATH=${CURRENT_DIRECTORY}/dist/${BINARY}
REPO=github.com/hashicorp/enos
GO_BUILD_TAGS=-tags osusergo,netgo
GO_LD_FLAGS=-ldflags="-extldflags=-static -X ${REPO}/internal/version.Version=${VERSION} -X ${REPO}/internal/version.GitSHA=${GIT_SHA}"
GO_GC_FLAGS?=
LINT_OUT_FORMAT?=colored-line-number
BUF_LINT_OUT_FORMAT?=github-actions
GORACE=GORACE=log_path=/tmp/enos-gorace.log
TEST_ACC=ENOS_ACC=1
TEST_ACC_EXT=ENOS_ACC=1 ENOS_EXT=1

.PHONY: default
default: build

.PHONY: all
all: generate build profile build-profile

.PHONY: generate
generate: generate-proto

.PHONY: generate-proto
generate-proto:
	pushd proto/hashicorp/enos/v1 && buf generate

.PHONY: build
build:
	CGO_ENABLED=0 go build ${GO_BUILD_TAGS} ${GO_LD_FLAGS} ${GO_GC_FLAGS} -o dist/${BINARY} ./command/enos

.PHONY: build-profile
build-profile:
	CGO_ENABLED=0 go build ${GO_BUILD_TAGS} ${GO_LD_FLAGS} ${GO_GC_FLAGS} -pgo=default.pgo -o dist/${BINARY} ./command/enos

.PHONY: build-race
build-race:
	${GORACE} go build -race ${GO_BUILD_TAGS} ${GO_LD_FLAGS} ${GO_GC_FLAGS} -o dist/${BINARY} ./command/enos

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

.PHONY: lint-fix
lint-fix: lint-fix-golang

.PHONY: lint-fix-golang
lint-fix-golang:
	golangci-lint run -v --out-format=$(LINT_OUT_FORMAT) --fix

.PHONY: lint-proto
lint-proto:
	buf lint --error-format=$(BUF_LINT_OUT_FORMAT)

.PHONY: fmt
fmt: fmt-golang fmt-proto fmt-terraform fmt-enos

.PHONY: fmt-check
fmt-check: fmt-check-golang fmt-check-proto fmt-check-terraform fmt-check-enos

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

.PHONY: fmt-terraform
fmt-terraform:
	terraform fmt -recursive acceptance/scenarios

.PHONY: fmt-check-terraform
fmt-check-terraform:
	terraform fmt -check -diff -recursive acceptance/scenarios

.PHONY: fmt-enos
fmt-enos:
	dist/enos fmt --recursive acceptance/scenarios

.PHONY: fmt-check-enos
fmt-check-enos:
	dist/enos fmt --check --diff --recursive acceptance/scenarios

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

.PHONY: profile
profile:
	@$(CURRENT_DIRECTORY)/tools/profile/profile.sh

.PHONY: version
version:
	@$(CURRENT_DIRECTORY)/.release/build-scripts/version.sh version/version.go
