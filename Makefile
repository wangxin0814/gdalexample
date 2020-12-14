.PHONY: build clean
VERSION := $(shell git describe --always |sed -e "s/^v//")


linux:export GOOS=linux
linux:export GOARCH=amd64
linux:export GODEBUG=cgocheck=0


build: clean
	@echo "Compiling source"
	@rm -rf build
	@mkdir -p build
	go build $(GO_EXTRA_BUILD_ARGS) -ldflags "-s -w -X main.version=$(VERSION)" -o build/gdalexample main.go

linux: build

microservice: linux
	@docker image rm wangxin0814/gdalexample:v1.0.0 | true
	@docker build  -t wangxin0814/gdalexample:v1.0.0 .
	@docker images wangxin0814/gdalexample:v1.0.0


clean:
	@echo "Cleaning up workspace"
	@rm -rf build