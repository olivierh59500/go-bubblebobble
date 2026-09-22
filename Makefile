GO ?= go

.PHONY: build run test check web android

build:
	$(GO) build -o bubblebobble .

run:
	$(GO) run .

android:
	./scripts/build-android.sh

test:
	$(GO) test ./...

check:
	$(GO) test -race ./...
	$(GO) vet ./...

web:
	mkdir -p dist/web
	GOOS=js GOARCH=wasm CGO_ENABLED=0 $(GO) build -o dist/web/game.wasm .
	cp "$$($(GO) env GOROOT)/lib/wasm/wasm_exec.js" dist/web/wasm_exec.js
	cp web/index.html dist/web/index.html
