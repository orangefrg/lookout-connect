.PHONY: build run test clean lint fmt
# Build the entire cmd directory
build:
	go build -o bin/lookout-connect ./cmd

# Build for different platforms
build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/lookout-connect-linux ./cmd

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/lookout-connect.exe ./cmd

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o bin/lookout-connect-mac ./cmd

# Run the application
run:
	go run cmd/main.go

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Lint the code
lint:
	golangci-lint run

# Format the code
fmt:
	go fmt ./...

# Install dependencies
deps:
	go mod tidy
	go mod download

# Generate documentation
docs:
	godoc -http=:6060
