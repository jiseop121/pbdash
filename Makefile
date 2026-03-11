GO ?= go
POCKETBASE_VERSION ?= 0.36.6
POCKETBASE_CACHE_DIR ?= .tmp/tools/pocketbase/$(POCKETBASE_VERSION)
POCKETBASE_DEFAULT_BIN := $(abspath $(POCKETBASE_CACHE_DIR)/pocketbase)
POCKETBASE_BIN ?= $(POCKETBASE_DEFAULT_BIN)
POCKETBASE_CHECKSUMS_URL := https://github.com/pocketbase/pocketbase/releases/download/v$(POCKETBASE_VERSION)/checksums.txt
PB_HTTP ?= 127.0.0.1:8090
PB_WORKDIR ?= .tmp/pocketbase-dev
PB_SUPERUSER_EMAIL ?= root@example.com
PB_SUPERUSER_PASSWORD ?= pass123456
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
SHA256_TOOL := $(shell if command -v shasum >/dev/null 2>&1; then echo shasum; elif command -v sha256sum >/dev/null 2>&1; then echo sha256sum; fi)

ifeq ($(UNAME_S),Darwin)
POCKETBASE_OS := darwin
else ifeq ($(UNAME_S),Linux)
POCKETBASE_OS := linux
else
POCKETBASE_OS := unsupported
endif

ifeq ($(UNAME_M),x86_64)
POCKETBASE_ARCH := amd64
else ifeq ($(UNAME_M),amd64)
POCKETBASE_ARCH := amd64
else ifeq ($(UNAME_M),arm64)
POCKETBASE_ARCH := arm64
else ifeq ($(UNAME_M),aarch64)
POCKETBASE_ARCH := arm64
else
POCKETBASE_ARCH := unsupported
endif

POCKETBASE_ASSET := pocketbase_$(POCKETBASE_VERSION)_$(POCKETBASE_OS)_$(POCKETBASE_ARCH).zip
POCKETBASE_DOWNLOAD_URL := https://github.com/pocketbase/pocketbase/releases/download/v$(POCKETBASE_VERSION)/$(POCKETBASE_ASSET)

.PHONY: test e2e pocketbase-bin pocketbase-superuser pocketbase-serve pb-su pb-serve release-tag release-brew

test:
	$(GO) test ./...

pocketbase-bin:
	@set -eu; \
	case "$(POCKETBASE_BIN)" in \
		*/*) if [ -x "$(POCKETBASE_BIN)" ]; then exit 0; fi ;; \
		*) if command -v "$(POCKETBASE_BIN)" >/dev/null 2>&1; then exit 0; fi ;; \
	esac; \
	if [ "$(POCKETBASE_BIN)" != "$(POCKETBASE_DEFAULT_BIN)" ]; then \
		echo "POCKETBASE_BIN is not executable or not found in PATH: $(POCKETBASE_BIN)"; \
		echo "Set POCKETBASE_BIN to a valid binary (path or command name) or unset it to use the auto-downloaded default."; \
		exit 1; \
	fi; \
	if [ "$(POCKETBASE_OS)" = "unsupported" ] || [ "$(POCKETBASE_ARCH)" = "unsupported" ]; then \
		echo "Unsupported platform for auto-downloading PocketBase: os=$(POCKETBASE_OS) arch=$(POCKETBASE_ARCH)"; \
		echo "Set POCKETBASE_BIN to an existing PocketBase binary."; \
		exit 1; \
	fi; \
	if ! command -v curl >/dev/null 2>&1; then \
		echo "curl is required to auto-download PocketBase."; \
		exit 1; \
	fi; \
	if ! command -v unzip >/dev/null 2>&1; then \
		echo "unzip is required to extract PocketBase."; \
		exit 1; \
	fi; \
	if [ -z "$(SHA256_TOOL)" ]; then \
		echo "shasum or sha256sum is required to verify the PocketBase download."; \
		exit 1; \
	fi; \
	mkdir -p "$(POCKETBASE_CACHE_DIR)"; \
	echo "Downloading PocketBase v$(POCKETBASE_VERSION) for $(POCKETBASE_OS)/$(POCKETBASE_ARCH)..."; \
	archive="$(POCKETBASE_CACHE_DIR)/$(POCKETBASE_ASSET)"; \
	checksums="$(POCKETBASE_CACHE_DIR)/checksums.txt"; \
	rm -f "$$archive" "$$checksums"; \
	curl -fsSL -o "$$archive" "$(POCKETBASE_DOWNLOAD_URL)"; \
	curl -fsSL -o "$$checksums" "$(POCKETBASE_CHECKSUMS_URL)"; \
	expected_sum=$$(awk '/ $(POCKETBASE_ASSET)$$/ {print $$1}' "$$checksums"); \
	if [ -z "$$expected_sum" ]; then \
		echo "Missing checksum for $(POCKETBASE_ASSET) in $(POCKETBASE_CHECKSUMS_URL)"; \
		exit 1; \
	fi; \
	if [ "$(SHA256_TOOL)" = "shasum" ]; then \
		actual_sum=$$(shasum -a 256 "$$archive" | awk '{print $$1}'); \
	else \
		actual_sum=$$(sha256sum "$$archive" | awk '{print $$1}'); \
	fi; \
	if [ "$$actual_sum" != "$$expected_sum" ]; then \
		echo "PocketBase checksum mismatch for $(POCKETBASE_ASSET)"; \
		echo "expected=$$expected_sum"; \
		echo "actual=$$actual_sum"; \
		exit 1; \
	fi; \
	unzip -oq "$$archive" -d "$(POCKETBASE_CACHE_DIR)"; \
	chmod +x "$(POCKETBASE_DEFAULT_BIN)"; \
	rm -f "$$archive" "$$checksums"; \
	if [ ! -x "$(POCKETBASE_DEFAULT_BIN)" ]; then \
		echo "PocketBase binary not found or not executable at $(POCKETBASE_DEFAULT_BIN)"; \
		exit 1; \
	fi

e2e: pocketbase-bin
	POCKETBASE_BIN=$(POCKETBASE_BIN) $(GO) test -tags=e2e ./e2e -v

pocketbase-superuser: pocketbase-bin
	mkdir -p $(PB_WORKDIR)
	cd $(PB_WORKDIR) && ($(POCKETBASE_BIN) superuser upsert $(PB_SUPERUSER_EMAIL) $(PB_SUPERUSER_PASSWORD) || $(POCKETBASE_BIN) superuser create $(PB_SUPERUSER_EMAIL) $(PB_SUPERUSER_PASSWORD))

pocketbase-serve: pocketbase-bin
	mkdir -p $(PB_WORKDIR)
	cd $(PB_WORKDIR) && $(POCKETBASE_BIN) serve --http=$(PB_HTTP)

pb-su: pocketbase-superuser

pb-serve: pocketbase-serve

release-tag:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Example: make release-tag VERSION=0.2.1"; exit 1; fi
	@if ! echo "$(VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$$'; then echo "VERSION must be semantic version without v prefix (e.g. 0.2.1)"; exit 1; fi
	./scripts/release_tag.sh "$(VERSION)"

release-brew:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Example: make release-brew VERSION=0.2.1"; exit 1; fi
	./scripts/release_brew_single_repo.sh --version "$(VERSION)" --github-repo "jiseop121/pbdash"
