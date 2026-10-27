# Development tasks. `make help` lists them.
# Ports are overridable when they are taken: make dev API_PORT=8090 WEB_PORT=5180
PG_PORT ?= 5439
API_PORT ?= 8080
WEB_PORT ?= 5173
export HUBLOT_PG_PORT := $(PG_PORT)
DEV_DB := postgres://hublot:hublot@localhost:$(PG_PORT)/hublot?sslmode=disable
TEST_DB := postgres://hublot:hublot@localhost:$(PG_PORT)/hublot_test?sslmode=disable
DEV_ENV := HUBLOT_CONFIG=deploy/dev/hublot.dev.yaml \
	HUBLOT_BASE_URL=http://localhost:$(WEB_PORT) \
	HUBLOT_LISTEN=:$(API_PORT) \
	HUBLOT_DATABASE_URL=$(DEV_DB) \
	HUBLOT_SMTP_HOST=localhost \
	HUBLOT_STORAGE_DIR=$(CURDIR)/.data/attachments \
	HUBLOT_FAKE_FORMS=$(CURDIR)/deploy/dev/forms \
	HUBLOT_WEB_DIR=

# web/node_modules can hold Go files: never use ./...
GO_PKGS := ./cmd/... ./internal/... ./web

.PHONY: help deps dev web-dev seed web build test test-db generate lint

help:
	@echo "deps      start Postgres and Mailpit (docker compose)"
	@echo "dev       run the API with the fake forge (and make web-dev in another terminal)"
	@echo "web-dev   run the Vite dev server, proxying /api to the API"
	@echo "seed      create the dev people and their grants"
	@echo "web       build the SPA into web/dist"
	@echo "build     build the single binary with the SPA embedded"
	@echo "test      run every test, including the database isolation suite"
	@echo "generate  regenerate the sqlc code"
	@echo "lint      go vet, gofmt and the web linters"

deps:
	docker compose up -d --wait postgres mailpit

dev:
	$(DEV_ENV) go run ./cmd/hublot serve

web-dev:
	cd web && HUBLOT_API=http://localhost:$(API_PORT) npx vite --port $(WEB_PORT) --strictPort

seed:
	$(DEV_ENV) go run ./cmd/hublot user -email enzo@acme.test -name Enzo -org-role owner
	$(DEV_ENV) go run ./cmd/hublot user -email marie@acme.test -name Marie
	$(DEV_ENV) go run ./cmd/hublot user -email paul@acme.test -name Paul
	$(DEV_ENV) go run ./cmd/hublot group -name Support -member marie@acme.test
	$(DEV_ENV) go run ./cmd/hublot grant -group Support -project boutique -role reporter
	$(DEV_ENV) go run ./cmd/hublot grant -group Support -project retours -role reporter
	$(DEV_ENV) go run ./cmd/hublot grant -email marie@acme.test -project outil-interne -role reporter
	$(DEV_ENV) go run ./cmd/hublot grant -email marie@acme.test -project site-vitrine -role viewer
	$(DEV_ENV) go run ./cmd/hublot grant -email paul@acme.test -project app-mobile -role contributor

web:
	cd web && npm ci && npm run build

build: web
	CGO_ENABLED=0 go build -tags embedweb -trimpath -o bin/hublot ./cmd/hublot

test-db: deps
	docker compose exec -T postgres psql -U hublot -tc "SELECT 1 FROM pg_database WHERE datname = 'hublot_test'" | grep -q 1 \
		|| docker compose exec -T postgres createdb -U hublot hublot_test

test: test-db
	HUBLOT_TEST_DATABASE_URL=$(TEST_DB) go test -p 1 $(GO_PKGS)

generate:
	sqlc generate

lint:
	go vet $(GO_PKGS)
	test -z "$$(gofmt -l cmd internal web/*.go)"
	cd web && npm run typecheck && npm run lint
