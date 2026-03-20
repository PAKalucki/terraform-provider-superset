default: fmt lint install generate

SUPERSET_ENDPOINT ?= http://127.0.0.1:8088
SUPERSET_USERNAME ?= admin
SUPERSET_PASSWORD ?= admin

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testenv-up:
	./scripts/testenv-up.sh

testenv-down:
	./scripts/testenv-down.sh

testenv-reset:
	./scripts/testenv-reset.sh

testenv-token:
	./scripts/testenv-token.sh

testacc: testenv-up
	TF_ACC=1 \
	SUPERSET_ENDPOINT=$(SUPERSET_ENDPOINT) \
	SUPERSET_USERNAME=$(SUPERSET_USERNAME) \
	SUPERSET_PASSWORD=$(SUPERSET_PASSWORD) \
	go test -v -cover -timeout 120m ./...

.PHONY: fmt lint test testacc build install generate testenv-up testenv-down testenv-reset testenv-token
