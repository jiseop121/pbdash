GO ?= go
POCKETBASE_BIN ?= pocketbase
PB_HTTP ?= 127.0.0.1:8090
PB_WORKDIR ?= .tmp/pocketbase-dev
PB_SUPERUSER_EMAIL ?= root@example.com
PB_SUPERUSER_PASSWORD ?= pass123456

.PHONY: test e2e pocketbase-superuser pocketbase-serve

test:
	$(GO) test ./...

e2e:
	POCKETBASE_BIN=$(POCKETBASE_BIN) $(GO) test -tags=e2e ./e2e -v

pocketbase-superuser:
	mkdir -p $(PB_WORKDIR)
	cd $(PB_WORKDIR) && ($(POCKETBASE_BIN) superuser create $(PB_SUPERUSER_EMAIL) $(PB_SUPERUSER_PASSWORD) || $(POCKETBASE_BIN) superuser upsert $(PB_SUPERUSER_EMAIL) $(PB_SUPERUSER_PASSWORD))

pocketbase-serve:
	mkdir -p $(PB_WORKDIR)
	cd $(PB_WORKDIR) && $(POCKETBASE_BIN) serve --http=$(PB_HTTP)
