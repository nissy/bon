# Makefile for Bon HTTP Router

.PHONY: help build test lint clean bench coverage fmt vet mod-tidy mod-download install

# デフォルトターゲット
.DEFAULT_GOAL := help

# ヘルプの表示
help: ## ヘルプを表示
	@echo "使用可能なコマンド:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ビルド
build: ## プロジェクトをビルド
	@echo "Building..."
	go build -v .

# テスト実行
test: ## すべてのテストを実行
	@echo "Running tests..."
	go test -v ./...

# テストの詳細実行（カバレッジ付き）
test-verbose: ## 詳細テストを実行（カバレッジ付き）
	@echo "Running verbose tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# ベンチマークテスト
bench: ## ベンチマークテストを実行
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# パフォーマンス比較ベンチマーク
bench-compare: ## パフォーマンス比較ベンチマークを実行
	@echo "Running performance comparison benchmarks..."
	go test -bench=BenchmarkMux -benchmem ./...

# Lintチェック（golangci-lintが必要）
lint: ## Lintチェックを実行
	@echo "Running lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1)
	golangci-lint run

# フォーマット
fmt: ## コードをフォーマット
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Vet実行
vet: ## go vetを実行
	@echo "Running go vet..."
	go vet ./...

# モジュール整理
mod-tidy: ## go mod tidyを実行
	@echo "Tidying modules..."
	go mod tidy

# 依存関係ダウンロード
mod-download: ## 依存関係をダウンロード
	@echo "Downloading dependencies..."
	go mod download

# インストール
install: ## バイナリをインストール
	@echo "Installing..."
	go install .

# カバレッジレポート生成
coverage: ## カバレッジレポートを生成
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# クリーンアップ
clean: ## 生成ファイルを削除
	@echo "Cleaning up..."
	go clean
	rm -f coverage.out coverage.html

# 開発環境セットアップ
dev-setup: ## 開発環境をセットアップ
	@echo "Setting up development environment..."
	go mod download
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# CI用のタスク（lint + test + build）
ci: lint test build ## CI用のタスクを実行

# すべてのチェック
check: fmt vet lint test ## すべてのチェックを実行

# リリース用ビルド
release: clean ## リリース用ビルド
	@echo "Building for release..."
	CGO_ENABLED=0 go build -ldflags="-w -s" -a -installsuffix cgo .

# パフォーマンステスト専用
perf: ## パフォーマンステストのみ実行
	@echo "Running performance tests..."
	go test -run=XXX -bench=. -benchtime=5s -benchmem ./...

# メモリプロファイル
profile-mem: ## メモリプロファイルを生成
	@echo "Generating memory profile..."
	go test -bench=BenchmarkMuxStaticRoute -benchmem -memprofile=mem.prof
	go tool pprof mem.prof

# CPUプロファイル
profile-cpu: ## CPUプロファイルを生成
	@echo "Generating CPU profile..."
	go test -bench=BenchmarkMuxStaticRoute -cpuprofile=cpu.prof
	go tool pprof cpu.prof