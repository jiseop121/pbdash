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

.PHONY: test e2e pocketbase-bin pocketbase-superuser pocketbase-serve pb-su pb-serve release-tag release-brew release-dry-run release-check

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
	@echo "release-brew는 deprecated되었습니다."
	@echo "태그 푸시만 하면 CI(GoReleaser)가 자동 처리합니다: make release-tag VERSION=x.y.z"
	@echo "로컬 테스트: make release-dry-run"
	@exit 1

release-check:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Example: make release-check VERSION=0.2.1"; exit 1; fi
	@if ! echo "$(VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$$'; then echo "VERSION must be semantic version without v prefix (e.g. 0.2.1)"; exit 1; fi
	@echo "==> 작업 트리 상태 확인"
	@if [ -n "$$(git status --porcelain)" ]; then echo "작업 트리가 깨끗하지 않습니다. 변경사항을 커밋하거나 stash하세요."; exit 1; fi
	@echo "==> 전체 테스트 실행"
	$(GO) test ./...
	@echo "==> 태그 중복 확인"
	@if git rev-parse -q --verify "refs/tags/v$(VERSION)" >/dev/null 2>&1; then \
		echo "태그 v$(VERSION)이 이미 로컬에 존재합니다."; exit 1; fi
	@echo "==> CHANGELOG 항목 확인"
	@if ! grep -q "$(VERSION)" CHANGELOG.md 2>/dev/null; then \
		echo "경고: CHANGELOG.md에 v$(VERSION) 항목이 없습니다."; fi
	@echo "✓ pre-release 검증 완료: v$(VERSION)"

release-dry-run:
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "goreleaser가 없습니다. 설치: brew install goreleaser"; exit 1; fi
	goreleaser release --snapshot --clean
	@echo "==> 빌드 아티팩트 존재 확인"
	@ls dist/pbdash-v*-darwin-amd64.tar.gz >/dev/null 2>&1 || (echo "오류: darwin-amd64 아티팩트가 없습니다."; exit 1)
	@ls dist/pbdash-v*-darwin-arm64.tar.gz >/dev/null 2>&1 || (echo "오류: darwin-arm64 아티팩트가 없습니다."; exit 1)
	@ls dist/pbdash-v*-linux-amd64.tar.gz >/dev/null 2>&1 || (echo "오류: linux-amd64 아티팩트가 없습니다."; exit 1)
	@ls dist/pbdash-v*-linux-arm64.tar.gz >/dev/null 2>&1 || (echo "오류: linux-arm64 아티팩트가 없습니다."; exit 1)
	@ls dist/pbdash-v*-checksums.txt >/dev/null 2>&1 || (echo "오류: 체크섬 파일이 없습니다."; exit 1)
	@echo "✓ dry-run 성공: 모든 아티팩트가 존재합니다."
