# Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.
SRCFILES := *.go go.sum go.mod

INSTALLPATH ?= /usr/local/bin
BINPATH:=$(abspath ./bin)

all: firectl

release: firectl firectl.sha256
	test $(shell git status --short | wc -l) -eq 0

firectl.sha256:
	sha256sum firectl > firectl.sha256

firectl: $(SRCFILES)
ifneq ($(STATIC_BINARY),)
	CGO_ENABLED=0 go build -installsuffix cgo -a
else
	go build
endif

build-in-docker:
	docker run --rm -v $(CURDIR):/firectl --workdir /firectl golang:1.14 make

test:
	go test -v ./...

GO_MINOR_VERSION = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)

$(BINPATH)/ltag:
	@if [ $(GO_MINOR_VERSION) -lt 16 ]; then \
		GOBIN=$(BINPATH) GO111MODULE=on go get -u github.com/kunalkushwaha/ltag@v0.2.3; \
	else \
		GOBIN=$(BINPATH) GO111MODULE=on go install github.com/kunalkushwaha/ltag@v0.2.3; \
	fi

$(BINPATH)/git-validation:
	@if [ $(GO_MINOR_VERSION) -lt 16 ]; then \
		GOBIN=$(BINPATH) GO111MODULE=on go get -u github.com/vbatts/git-validation@v1.1.0; \
	else \
		GOBIN=$(BINPATH) GO111MODULE=on go install github.com/vbatts/git-validation@v1.1.0; \
	fi

$(BINPATH)/golangci-lint:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(BINPATH) v1.46.2
	$(BINPATH)/golangci-lint --version

lint: $(BINPATH)/ltag $(BINPATH)/git-validation $(BINPATH)/golangci-lint
	$(BINPATH)/ltag -v -t ./.headers -check
	$(BINPATH)/git-validation -q -run DCO,short-subject -range HEAD~5..HEAD
	$(BINPATH)/golangci-lint run

clean:
	go clean

install:
	install -o root -g root -m755 -t $(INSTALLPATH) firectl

.PHONY: all clean install build-in-docker test lint release
