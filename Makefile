.PHONY: build run test vet fmt check

BINARY=build/arena

build:
	go generate ./... 2>/dev/null || true
	mkdir -p build
	go build -o $(BINARY) ./cmd/arena

run: build
	./$(BINARY) run -c config/config.json

leaderboard: build
	./$(BINARY) leaderboard -c config/config.json

test:
	# jangan dijalankan di awal sesuai aturan - cek manual dulu
	@echo "tests are written but not executed (per aturan: cek dulu)"
	@echo "run: go test ./... -count=1"

vet:
	go vet ./pkg/... ./cmd/...

fmt:
	gofmt -w ./pkg ./cmd

check: vet
	go test ./... -count=1

clean:
	rm -rf build/
