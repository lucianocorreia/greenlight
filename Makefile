## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

## run/api: run the cmd/api application
.PHONY: run
run:
	@go run ./cmd/api

## db/migrations/up: apply all up database migrations
.PHONY: up
up: confirm
	@echo 'Running up migrations...'
	migrate -path=./migrations -database="postgres://postgres:postgres@localhost/greenlight?sslmode=disable" up

.PHONY: down
down:
	@echo 'Running up migrations...'
	migrate -path=./migrations -database="postgres://postgres:postgres@localhost/greenlight?sslmode=disable" down

## db/migrations/new name=$1: create a new database migration
.PHONY: migrations
migration:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/psql: connect to the database using psql
.PHONY: psql
psql:
	/Applications/Postgres.app/Contents/Versions/16/bin/psql -p5432 "greenlight"

current_time = $(shell date --iso-8601=seconds) 
linker_flags = '-s -X main.buildTime=${current_time}'
## build/api: build the cmd/api application
.PHONY: build/api 
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags=${linker_flags} -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./bin/linux_amd64/api ./cmd/api

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...' 
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

