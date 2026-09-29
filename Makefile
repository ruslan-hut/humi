BIN     := bin
WEB_OUT := web/dist/web/browser

.PHONY: all build web run test lint clean node invite bundle

all: web build

build:
	go build -o $(BIN)/humi ./cmd/server
	go build -o $(BIN)/nodectl ./cmd/nodectl
	go build -o $(BIN)/userctl ./cmd/userctl

web:
	cd web && npm ci && npx ng build

run: build
	./$(BIN)/humi -conf config.yml -log .

# Register a node and print its ingest token once:
#   make node SLUG=bedroom NAME=Bedroom
node: build
	./$(BIN)/nodectl -conf config.yml -slug $(SLUG) -name "$(NAME)" -interval $(or $(INTERVAL),900)

# Print a single-use join link; the first admin comes from here:
#   make invite ROLE=admin
invite: build
	./$(BIN)/userctl -conf config.yml -invite $(or $(ROLE),viewer)

# Everything the server needs, for deploy/install.sh. Add humi.db next to it
# to seed a fresh server with an existing database.
bundle: web
	rm -rf $(BIN)/bundle && mkdir -p $(BIN)/bundle
	for arch in amd64 arm64; do \
		GOOS=linux GOARCH=$$arch CGO_ENABLED=0 go build -o $(BIN)/bundle/bin/linux-$$arch/humi ./cmd/server && \
		GOOS=linux GOARCH=$$arch CGO_ENABLED=0 go build -o $(BIN)/bundle/bin/linux-$$arch/nodectl ./cmd/nodectl && \
		GOOS=linux GOARCH=$$arch CGO_ENABLED=0 go build -o $(BIN)/bundle/bin/linux-$$arch/userctl ./cmd/userctl || exit 1; \
	done
	cp -R $(WEB_OUT) $(BIN)/bundle/web
	cp deploy/config.yml deploy/humi.service deploy/humi.nginx deploy/install.sh deploy/receive.sh $(BIN)/bundle/
	COPYFILE_DISABLE=1 tar --no-xattrs -C $(BIN) -czf $(BIN)/humi-bundle.tgz bundle

test:
	go test -race ./...

lint:
	gofmt -l .
	go vet ./...

clean:
	rm -rf $(BIN) web/dist
