export GO111MODULE ?= on
export GOPROXY ?= https://proxy.golang.org
export GOSUMDB ?= sum.golang.org
CGO			?= 0
CLI_BINARY  = fn

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

LOCAL_ARCH := $(shell uname -m)
ifeq ($(LOCAL_ARCH),x86_64)
	TARGET_ARCH_LOCAL = amd64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),armv8)
	TARGET_ARCH_LOCAL = arm64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),aarch64)
	TARGET_ARCH_LOCAL = arm64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 4),armv)
	TARGET_ARCH_LOCAL = arm
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),arm64)
    TARGET_ARCH_LOCAL = arm64
else
	TARGET_ARCH_LOCAL = amd64
endif
export GOARCH ?= $(TARGET_ARCH_LOCAL)

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
   TARGET_OS_LOCAL = linux
   GOLANGCI_LINT:=golangci-lint
   export ARCHIVE_EXT = .tar.gz
else ifeq ($(LOCAL_OS),Darwin)
   TARGET_OS_LOCAL = darwin
   GOLANGCI_LINT:=golangci-lint
   export ARCHIVE_EXT = .tar.gz
else
   TARGET_OS_LOCAL ?= windows
   BINARY_EXT_LOCAL = .exe
   GOLANGCI_LINT:=golangci-lint.exe
   export ARCHIVE_EXT = .zip
endif
export GOOS ?= $(TARGET_OS_LOCAL)
export BINARY_EXT ?= $(BINARY_EXT_LOCAL)

TEST_OUTPUT_FILE ?= test_output.json

# Use the variable H to add a header (equivalent to =>) to informational output
H = $(shell printf "\033[34;1m=>\033[0m")

ifeq ($(origin DEBUG), undefined)
  BUILDTYPE_DIR:=release
else ifeq ($(DEBUG),0)
  BUILDTYPE_DIR:=release
else
  BUILDTYPE_DIR:=debug
  GCFLAGS:=-gcflags="all=-N -l"
  $(info $(H) Build with debugger information)
endif

LDFLAGS="-s -w -X 'main.goversion=$(shell go version)'"

################################################################################
# Go build details                                                             #
################################################################################
OUT_DIR := ./dist

BINS_OUT_DIR := $(OUT_DIR)

################################################################################
# Target: build                                                                #
################################################################################
.PHONY: build
build: fmt vet $(CLI_BINARY)

$(CLI_BINARY):
	CGO_ENABLED=$(CGO) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GCFLAGS) -ldflags $(LDFLAGS) \
	-o $(BINS_OUT_DIR)/$(CLI_BINARY)_$(GOOS)_$(GOARCH)$(BINARY_EXT) cmd/main.go;

################################################################################
# Target: lint                                                                 #
################################################################################
.PHONY: lint
lint:
	$(GOLANGCI_LINT) run --timeout=20m

fmt: goimports ## Run go fmt && goimports against code.
	go fmt ./...
	$(GOIMPORTS) -w cmd/ pkg/ testdata/

GOIMPORTS=$(shell pwd)/bin/goimports

goimports:
	$(call go-get-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports@v0.1.7)

vet: ## Run go vet against code.
	go vet ./...

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install -v $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
