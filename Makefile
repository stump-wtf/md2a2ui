.PHONY: test lint check clean

test:
	go test -race -cover ./...

lint:
	go vet ./...
	@test -z "$$(gofumpt -l . 2>/dev/null)" || (echo "gofumpt needs to run:"; gofumpt -l .; exit 1)

check: lint test

clean:
	rm -f coverage.txt
