BIN := bin/qrt

.PHONY: build test demo clean

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/qrt

test:
	go test ./...

demo: build
	rm -rf .qratum
	cat fixtures/claude-code/hook-session-end.json | ./$(BIN) hook claude-code
	./$(BIN) daemon run-once
	./$(BIN) sessions list
	test -n "$$(find .qratum/events -name '*.json' -print -quit)"
	test -n "$$(find .qratum/sessions -name '*.normalized.json' -print -quit)"
	test -n "$$(find .qratum/redacted -name '*.redacted.json' -print -quit)"
	test -n "$$(find .qratum/evidence -name '*.evidence.json' -print -quit)"
	test -n "$$(find .qratum/reviews -name '*.review.json' -print -quit)"
	test -n "$$(find .qratum/reports -name '*.html' -print -quit)"
	test -n "$$(find .qratum/exports -name '*.adp.jsonl' -print -quit)"
	@echo "Generated artifacts:"
	@find .qratum -type f | sort

clean:
	rm -rf bin .qratum
