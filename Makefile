.PHONY: generate

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

.PHONY: generate migrate

migrate:
	set -a; . ./.env; set +a; \
	go tool goose -dir migrations postgres "$$DATABASE_URL" up

.PHONY: generate migrate run test

run:
	set -a; . ./.env; set +a; \
	go run ./cmd/trip-service

test:
	go test -race -cover ./...

.PHONY: test-integration

test-integration:
	set -a; . ./.env; set +a; \
	go test -race -tags=integration ./...

.PHONY: lint

lint:
	golangci-lint run