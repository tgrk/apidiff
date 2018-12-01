
.PHONY: default
default:
	@echo "The following targets are available:"
	@echo "  deps        - installs all necessary application dependencies and development tools"
	@echo "  test        - runs the test suite"
	@echo "  verboseTest - runs the test suite printing all the output"
	@echo "  format      - reformats the code"
	@echo "  lint        - invokes gometalinter"
	@echo "  cover       - runs the test suite and shows coverage"
	@echo "  docs        - runs godoc server on :6060"
	@echo "  clean       - cleans the build and test cache"
	@echo "  build       - builds executable"
	@echo "  runs        - runs executable"

.PHONY: devDeps
devDeps:
	go get -u github.com/alecthomas/gometalinter
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/kardianos/govendor
	go get -u github.com/joho/godotenv/cmd/godotenv

.PHONY: deps
deps: devDeps
	govendor sync -v
	govendor list

.PHONY: test
test:
	go test -count 1 -p 1 ./...

.PHONY: verboseTest
verboseTest:
	go test -v -count 1 -p 1 ./...

.PHONY: lint
lint:
	gometalinter

.PHONY: cover
cover:
	go test -coverprofile fmt

.PHONY: format
format:
	govendor fmt +local

.PHONY: docs
docs:
	godoc -http=:6060

.PHONY: clean
clean:
	go clean -cache

.PHONY: build
build:
	go build cmd/apidiff/apidiff.go

.PHONY: run
run:
	go run cmd/apidiff/apidiff.go

.PHONY: ciDeps
ciDeps: deps
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	go get github.com/jstemmer/go-junit-report

.PHONY: ciTest
ciTest:
	go test -p 1 -coverprofile=coverage.txt -covermode=atomic ./...
