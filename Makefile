# Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
EXECUTABLE                  := nsxt-lb-provider-manager

REGISTRY                    := eu.gcr.io/gardener-project
IMAGE_PREFIX                := $(REGISTRY)/test
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
VERSION                     := $(shell bash -c 'source $(HACK_DIR)/common.sh && echo $$VERSION')
LD_FLAGS                    := "-w -X github.com/gardener/gardener-extensions/pkg/version.Version=$(IMAGE_TAG)"
VERIFY                      := true


.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	   -mod=vendor \
	   -o bin/rel/$(EXECUTABLE) \
	   -ldflags "-X main.Version=$(VERSION)" \
	    ./cmd/$(EXECUTABLE)

.PHONY: build-local
build-local:
	go build \
	    -mod=vendor \
	    -o $(EXECUTABLE) \
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
