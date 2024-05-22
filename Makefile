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
	docker run --rm -v $(CURDIR):/firectl --workdir /firectl golang:1.22 make

test:
	go test -v ./...

deps = \
	$(BINPATH)/golangci-lint \
	$(BINPATH)/git-validation \
	$(BINPATH)/ltag

lint: $(deps)
	$(BINPATH)/ltag -t ./.headers -excludes "tools $(SUBMODULES)" -check -v
	$(BINPATH)/git-validation -run DCO,short-subject -range HEAD~5..HEAD
	$(BINPATH)/golangci-lint run

$(BINPATH)/golangci-lint:
	GOBIN=$(BINPATH) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.58.2
	$(BINPATH)/golangci-lint --version

$(BINPATH)/git-validation:
	GOBIN=$(BINPATH) go install github.com/vbatts/git-validation@v1.2.0

$(BINPATH)/ltag:
	GOBIN=$(BINPATH) go install github.com/kunalkushwaha/ltag@v0.2.5

clean:
	go clean
	rm -rf $(BINPATH)

install:
	install -o root -g root -m755 -t $(INSTALLPATH) firectl

.PHONY: all clean install build-in-docker test lint release
