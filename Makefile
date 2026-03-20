.PHONY: help build test run demo clean fmt lint

help:
	@echo "0G Memory Hub - Available Commands"
	@echo "=================================="
	@echo "make build       - Build the project"
	@echo "make test        - Run tests"
	@echo "make run         - Run the CLI"
	@echo "make demo        - Run end-to-end demo"
	@echo "make clean       - Clean build artifacts"
	@echo "make fmt         - Format code"
	@echo "make lint        - Run clippy linter"
	@echo "make check       - Check code without building"

build:
	@echo "🔨 Building project..."
	cargo build --release

test:
	@echo "🧪 Running tests..."
	cargo test --release

run:
	@echo "▶️  Running CLI..."
	cargo run --release -- --help

demo:
	@echo "🚀 Running end-to-end demo..."
	bash demo.sh

clean:
	@echo "🧹 Cleaning build artifacts..."
	cargo clean
	rm -f demo_memory.json demo_memory_restored.json

fmt:
	@echo "📝 Formatting code..."
	cargo fmt

lint:
	@echo "🔍 Running clippy..."
	cargo clippy --all-targets --all-features -- -D warnings

check:
	@echo "✓ Checking code..."
	cargo check

upload:
	@echo "📤 Uploading file to 0G Storage..."
	cargo run --release -- upload $(FILE) --replicas 2

download:
	@echo "📥 Downloading from 0G Storage..."
	cargo run --release -- download $(CID) --output $(OUTPUT) --verify

set-pointer:
	@echo "🔗 Setting memory pointer on-chain..."
	cargo run --release -- set-pointer $(AGENT) $(CID)

get-pointer:
	@echo "🔍 Getting memory pointer..."
	cargo run --release -- get-pointer $(AGENT)

get-history:
	@echo "📚 Getting memory history..."
	cargo run --release -- get-history $(AGENT)

.DEFAULT_GOAL := help
