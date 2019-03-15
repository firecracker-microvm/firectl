# Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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
SRCFILES := *.go

all: firectl

firectl: $(SRCFILES)
	go build -o firectl

release: $(SRCFILES)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		    -a \
		    -installsuffix cgo \
		    -ldflags "-s" \
		    -gcflags=all=-trimpath=${TRIMPATH} \
		    -asmflags=all=-trimpath=${TRIMPATH} \
		    -o firectl

test:
	go test -v ./...

lint:
	golint $(SRCFILES)

clean:
	go clean

.PHONY: all clean
