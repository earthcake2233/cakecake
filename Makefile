# MiniBili Makefile -- cross-platform build & test entry.
#   Linux/macOS: make <target>
#   Windows:     make <target>   (GNU Make required, uses cmd.exe)

ifeq ($(OS),Windows_NT)
SHELL := cmd.exe
.SHELLFLAGS := /c
endif

.PHONY: all test test-backend test-frontend coverage coverage-backend coverage-frontend clean help

GO = go
NPM = npm
VUE_DIR = cakecake-vue/bilibili-vue

all: test

# -- Backend --------------------------------------------------------

test-backend:
	$(GO) test -tags=integration -count=1 -timeout 150s ./internal/...

coverage-backend:
	$(GO) test -tags=integration -cover -coverprofile=coverage.out -covermode=count -count=1 -timeout 150s ./internal/...

# -- Frontend -------------------------------------------------------

test-frontend:
	cd $(VUE_DIR) && $(NPM) test

coverage-frontend:
	cd $(VUE_DIR) && $(NPM) run coverage

# -- Combined -------------------------------------------------------

test: test-backend test-frontend

coverage: coverage-backend coverage-frontend

# -- Cleanup --------------------------------------------------------

clean:
	python clean.py



# Clean Go build cache from C: drive (prevents C: from filling up)
clean-go-cache:
	python scripts/clean_go_cache.py --system-only
# -- Build ---------------------------------------------------------

# Build Linux amd64 binary (cross-compile from any platform)
build-linux:
	$(GO) clean -cache
	$(GO) build -ldflags="-s -w" -o mini-bili-linux ./cmd/mini-bili
	python scripts/clean_go_cache.py --system-only

# Export env vars for cross-compilation (must be env, not Make vars)
build-linux: export GO111MODULE := on
build-linux: export GOOS := linux
build-linux: export GOARCH := amd64
build-linux: export GOCACHE := $(CURDIR)/.gocache
build-linux: export GOTMPDIR := $(CURDIR)/.gotmp
build-linux: export GOPATH = $(TEMP)/gopath-clean

# Build frontend production bundle
build-frontend:
	cd $(VUE_DIR) && $(NPM) install
ifeq ($(OS),Windows_NT)
	cd $(VUE_DIR) && copy /Y .env.production.example .env.production
else
	cd $(VUE_DIR) && cp .env.production.example .env.production
endif
	cd $(VUE_DIR) && $(NPM) run build

# -- Help -----------------------------------------------------------

help:
	$(info   make build-linux      Cross-compile Go binary for Linux amd64)
	$(info   make build-frontend    Build Vue production bundle)
	$(info Usage:)
	$(info   make test             Run all tests (backend + frontend))
	$(info   make test-backend     Run Go tests only)
	$(info   make test-frontend    Run Vue tests only)
	$(info   make coverage         Run all tests with coverage)
	$(info   make coverage-backend Go coverage (output: coverage.out))
	$(info   make coverage-frontend Vue coverage)
	$(info   make clean            Remove temp/coverage artifacts)
