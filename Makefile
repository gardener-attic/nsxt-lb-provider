EXECUTABLE                  := nsxt-lb-provider-manager

REGISTRY                    := eu.gcr.io/gardener-project
IMAGE_PREFIX                := $(REGISTRY)/test
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
VERSION                     := $(shell bash -c 'source $(HACK_DIR)/common.sh && echo $$VERSION')
LD_FLAGS                    := "-w -X github.com/gardener/gardener-extensions/pkg/version.Version=$(IMAGE_TAG)"
VERIFY                      := true


.PHONY: build-local build release test

build:
	GOOS=linux GOARCH=amd64 go build -mod=vendor -o $(EXECUTABLE) \
	    -ldflags "-X main.Version=$(VERSION)-$(shell git rev-parse HEAD)"\
	    ./cmd/$(EXECUTABLE)

build-local:
	go build -mod=vendor -o $(EXECUTABLE) \
	    -ldflags "-X main.Version=$(VERSION)-$(shell git rev-parse HEAD)"\
	    ./cmd/$(EXECUTABLE)

.PHONY: docker-image
docker-image:
	@docker build --build-arg VERIFY=$(VERIFY) -t $(IMAGE_PREFIX)/$(EXECUTABLE):$(VERSION) -t $(IMAGE_PREFIX)/$(EXECUTABLE):latest -f Dockerfile --target $(EXECUTABLE) .

.PHONY: check
check:
	@./hack/check.sh

.PHONY: format
format:
	@./hack/format.sh

.PHONY: test
test:
	@./hack/test.sh

.PHONY: verify
verify: check test format

.PHONY: install
install:
	@./hack/install.sh

.PHONY: all
ifeq ($(VERIFY),true)
all: verify install
else
all: install
endif
