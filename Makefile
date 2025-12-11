# ******************************************************************************
# IBM Cloud Kubernetes Service, 5737-D43
# (C) Copyright IBM Corp. 2021, 2025 All Rights Reserved.
#
# SPDX-License-Identifier: Apache2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ******************************************************************************
export
GO_PACKAGES:=$(shell go list ./... )
SH_FILES=$(shell find . -type f -name '*.sh')

# Calicoctl
CALICOCTL_VERSION?=3.27.5
CALICOCTL_URL:="https://github.com/projectcalico/calico/releases/download/v${CALICOCTL_VERSION}/calicoctl-linux-amd64"

# golangci_lint
GOLANGCI_LINT_VERSION := 2.7.2
GOLANGCI_LINT_EXISTS := $(shell golangci-lint --version 2>/dev/null)

.PHONY: all
all: fmt lint lint-sh vet strongswan

.PHONY: fmt
fmt:
ifdef GOLANGCI_LINT_EXISTS
	golangci-lint fmt -v
else
	@echo "golangci-lint is not installed"
	exit 1
endif

.PHONY: lint
lint:
ifdef GOLANGCI_LINT_EXISTS
	echo "Running gosec"
	golangci-lint run -v --timeout 5m
else
	@echo "golangci-lint is not installed"
	exit 1
endif

.PHONY: lint-sh
lint-sh:
	shellcheck -x -V
	shellcheck ${SH_FILES}

.PHONY: vet
vet:
	go vet ${GO_PACKAGES}

.PHONY: build
build:
	cd strongswan; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o strongswan -ldflags '-s -w' .

.PHONY: package
package:
	cd helm; \
	helm lint strongswan; \
	helm package strongswan; \
	cp strongswan*.tgz packages; \
	rm strongswan*.tgz

.PHONY: strongswan
strongswan: build calicoctl
	cd strongswan; \
	docker build -t strongswan -f Dockerfile . ; \
	docker images | grep strongswan

.PHONY: calicoctl
calicoctl:
	# downloading calicoctl binary
	curl -L ${CALICOCTL_URL} --output ./strongswan/calicoctl --fail
	chmod 755 ./strongswan/calicoctl

.PHONY: clean
clean:
	rm -f strongswan/calicoctl
	rm -f strongswan/strongswan
