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

build:
	CGO_ENABLED=0 go build ${GO_BUILD_TAGS} ${GO_LD_FLAGS} ${GO_GC_FLAGS} -o dist/${BINARY} ./command/enos

build-race:
	${GORACE} go build -race ${GO_BUILD_TAGS} ${GO_LD_FLAGS} ${GO_GC_FLAGS} -o dist/${BINARY} ./command/enos

test:
	${GORACE} go test -race ./... -v $(TESTARGS) -timeout=5m -parallel=4

test-acc: build-race
	${TEST_ACC} ${GORACE} ENOS_BINARY_PATH=${BUILD_BINARY_PATH} go test -race ./... -v $(TESTARGS) -timeout 120m

test-acc-ext: build-race
	${TEST_ACC_EXT} ${GORACE} ENOS_BINARY_PATH=${BUILD_BINARY_PATH} go test -race ./... -v $(TESTARGS) -timeout 120m

lint:
	golangci-lint run -v

fmt:
	gofumpt -w -l .

clean:
	rm -rf dist enos

version:
	@$(CURRENT_DIRECTORY)/.release/build-scripts/version.sh version/version.go

.PHONY: version
