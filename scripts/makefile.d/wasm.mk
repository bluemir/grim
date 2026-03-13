##@ wasm

GOROOT_PATH := $(shell go env GOROOT)

wasm: assets/bundle/wasm/$(APP_NAME).wasm
assets/bundle/wasm/$(APP_NAME).wasm: $(GO_SOURCES)
	@mkdir -p $(dir $@)
	GOOS=js GOARCH=wasm go build -v  \
		-trimpath \
		-ldflags "\
			-X '$(IMPORT_PATH)/internal/buildinfo.AppName=$(APP_NAME)' \
			-X '$(IMPORT_PATH)/internal/buildinfo.Version=$(VERSION)' \
			-X '$(IMPORT_PATH)/internal/buildinfo.BuildTime=$(shell go run scripts/tools/date/main.go)' \
		" \
		$(OPTIONAL_BUILD_ARGS) \
		-o $@ ./wasm

## wasm_exec.js (Go WASM glue code for browser)
assets/bundle/wasm/wasm_exec.js: $(GOROOT_PATH)/lib/wasm/wasm_exec.js
	@mkdir -p $(dir $@)
	cp $< $@

OPTIONAL_CLEAN += assets/bundle/wasm
