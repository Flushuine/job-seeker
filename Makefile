ifneq (,$(wildcard .env))
include .env
export
endif
.PHONY: setup db api worker web build test test-integration migrate
setup:
	go mod download
	npm --prefix web install
db:
	docker compose up -d postgres
api:
	go run ./cmd/api
worker:
	go run ./cmd/worker
web:
	npm --prefix web run dev
build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	npm --prefix web run build
test:
	go test ./... -timeout=60s
test-integration:
	@test -n "$(TEST_DATABASE_URL)" || (echo 'Set TEST_DATABASE_URL to an isolated test database'; exit 1)
	go test ./... -count=1 -timeout=60s
migrate:
	go run ./cmd/migrate

.PHONY: db-native db-native-stop
db-native:
	sh scripts/postgres-local.sh start
db-native-stop:
	sh scripts/postgres-local.sh stop
