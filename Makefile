# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=gnetwork
BINARY_UNIX=$(BINARY_NAME)_unix
all: test build
protos:
	cd common/protobuff/protos && PATH=$(PATH):$(GOPATH)/bin protoc --go_out=$(PKGMAP):.. *.proto
build:
	$(GOBUILD) -o $(BINARY_NAME) -v .
	chmod 775 $(BINARY_NAME)
docker:
	export DOCKER_BUILDKIT=0
	docker build .
	chmod 775 $(BINARY_NAME)
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
run:
	protos
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)
deps:
	$(GOGET) github.com/markbates/goth
	$(GOGET) github.com/markbates/pop
.PHONY: mocks
mocks:
	/usr/local/bin/mockery --all