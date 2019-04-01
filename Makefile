# Copyright 2018-2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
# 	http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

SUBDIRS:=agent runtime snapshotter examples
export INSTALLROOT?=/usr/local
export STATIC_AGENT

GOPATH:=$(shell go env GOPATH)
UID:=$(shell id -u)

all: $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@

proto:
	proto/generate.sh

clean:
	for d in $(SUBDIRS); do $(MAKE) -C $$d clean; done
	$(MAKE) -C runc clean
	rm -f *stamp
	$(MAKE) -C image-builder clean-in-docker

distclean: clean
	docker rmi runc-builder:latest
	$(MAKE) -C image-builder distclean

deps:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOPATH)/bin v1.12.3
	GO111MODULE=off go get -u github.com/vbatts/git-validation
	GO111MODULE=off go get -u github.com/kunalkushwaha/ltag

lint:
	ltag -t ./.headers -excludes runc -check -v
	git-validation -run DCO,dangling-whitespace,short-subject -range HEAD~20..HEAD
	golangci-lint run

runc-builder: runc-builder-stamp

runc-builder-stamp: tools/docker/Dockerfile.runc-builder
	cd tools/docker && docker build -t runc-builder:latest -f Dockerfile.runc-builder .
	touch $@

runc: runc/runc

runc/runc: runc-builder-stamp
	docker run --rm -it --user $(UID) \
		--volume $(PWD)/runc:/gopath/src/github.com/opencontainers/runc \
		--volume $(PWD)/deps:/target \
		-e HOME=/tmp \
		-e GOPATH=/gopath \
		--workdir /gopath/src/github.com/opencontainers/runc \
		runc-builder:latest \
		make runc

image: runc/runc agent
	mkdir -p image-builder/files_ephemeral/usr/local/bin
	cp runc/runc image-builder/files_ephemeral/usr/local/bin
	cp agent/agent image-builder/files_ephemeral/usr/local/bin
	touch image-builder/files_ephemeral
	$(MAKE) -C image-builder all-in-docker

install:
	for d in $(SUBDIRS); do $(MAKE) -C $$d install; done

.PHONY: all $(SUBDIRS) image clean proto deps lint install runc-builder runc
