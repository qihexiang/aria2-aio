.PHONY: all frontend build clean run-dev

BINARY_NAME := aria2-aio
FRONTEND_DIR := frontend

all: frontend build

frontend:
	cd $(FRONTEND_DIR) && npm install && npm run build

build:
	CGO_ENABLED=0 go build -o $(BINARY_NAME) ./cmd/aria2-aio/

clean:
	rm -f $(BINARY_NAME)
	rm -rf ui/dist
	cd $(FRONTEND_DIR) && rm -rf node_modules

run-dev: build
	./$(BINARY_NAME) --dev --port 8080

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 ./cmd/aria2-aio/

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME)-linux-arm64 ./cmd/aria2-aio/

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 ./cmd/aria2-aio/

build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe ./cmd/aria2-aio/