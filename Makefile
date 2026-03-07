BINARY := gh-dashboard
GO     := go

.PHONY: build run lint vet tidy clean

build:
	$(GO) build -o $(BINARY) .

run: build
	./$(BINARY)

lint:
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

clean:
	rm -f $(BINARY)
