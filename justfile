# Justfile for kata-cli

default: build

# Build the kata-cli binary
build:
	go build -o kata-cli ./cmd/kata-cli

# Run the kata-cli application directly
run:
	go run ./cmd/kata-cli

# Clean build artifacts
clean:
	rm -rf kata-cli

# Tidy go module dependencies
tidy:
	go mod tidy
