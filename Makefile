BIN := bin/qrt

.PHONY: build test demo clean

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/qrt

test:
	go test ./...

demo: build
	sh scripts/demo.sh ./$(BIN)

clean:
	rm -rf bin .qratum
