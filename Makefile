BIN     := bin
WEB_OUT := web/dist/web/browser

.PHONY: all build web run test lint clean node

all: web build

build:
	go build -o $(BIN)/humi ./cmd/server
	go build -o $(BIN)/nodectl ./cmd/nodectl

web:
	cd web && npm ci && npx ng build

run: build
	./$(BIN)/humi -conf config.yml -log .

# Register a node and print its ingest token once:
#   make node SLUG=bedroom NAME=Bedroom
node: build
	./$(BIN)/nodectl -conf config.yml -slug $(SLUG) -name "$(NAME)" -interval $(or $(INTERVAL),900)

test:
	go test -race ./...

lint:
	gofmt -l .
	go vet ./...

clean:
	rm -rf $(BIN) web/dist
