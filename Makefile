BINARIES = $(wildcard bin/*/*)
INSTALL_DIR ?=/usr/local/bin/

BINARIE_NAME = disk-image-install
BUILD_DIR = ./cmd

TAG = $(shell git describe --tags)
LDFLAGS = "-X 'main.version=$(TAG)'"

build-linux-amd64: clean
	GOOS=linux GOARCH=amd64 go build -ldflags=$(LDFLAGS) -o .bin/linux-amd64/$(BINARIE_NAME)  $(BUILD_DIR)

build:
	go build -ldflags=$(LDFLAGS) -o .bin/$(BINARIE_NAME) $(BUILD_DIR) 

install: build
	sudo install -Dm755 .bin/$(BINARIE_NAME)  $(INSTALL_DIR)/$(BINARIE_NAME)

compress-all-binaries: build-all-binaries
	for f in $(BINARIES); do      \
        tar czf $$f.tar.gz $$f;    \
    done
	@rm $(BINARIES)

test: $(SOURCES)
	@go vet ./...
	@go test -v ./...
	@test -z $(shell gofmt -s -l . | tee /dev/stderr) || (echo "[ERROR] Fix formatting issues with 'gofmt'" && exit 1)

.PHONY: clean
clean:
	rm -Rf .bin; rm -Rf *.tar.gz
