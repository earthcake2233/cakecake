# MiniBili Makefile -- cross-platform build & test entry.
#   Linux/macOS: make <target>
#   Windows:     make <target>   (GNU Make required)

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

# -- Help -----------------------------------------------------------

help:
	$(info Usage:)
	$(info   make test             Run all tests (backend + frontend))
	$(info   make test-backend     Run Go tests only)
	$(info   make test-frontend    Run Vue tests only)
	$(info   make coverage         Run all tests with coverage)
	$(info   make coverage-backend Go coverage (output: coverage.out))
	$(info   make coverage-frontend Vue coverage)
	$(info   make clean            Remove temp/coverage artifacts)
