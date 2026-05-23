.PHONY: build build-all clean tag version

NAME    := neodlp
VERSION := $(shell cat .version)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags="-X neodlp/internal/version.Commit=$(COMMIT) \
                     -X neodlp/internal/version.PublisherName=rkriad585 \
                     -X neodlp/internal/version.PublisherEmail=rkriad585@gmail.com"

build:
	go build $(LDFLAGS) -o $(NAME) .

build-all:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(NAME)-windows-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o bin/$(NAME)-windows-arm64.exe .
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/$(NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o bin/$(NAME)-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o bin/$(NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o bin/$(NAME)-darwin-arm64 .

clean:
	rm -rf bin
	rm -f $(NAME) $(NAME).exe

tag:
	git tag $(VERSION)
	git push origin $(VERSION)

version:
	@echo $(VERSION)
