.PHONY: migrate-up migrate-down migrate-version migrate-steps migrate-create lint lint-fix lint-fast

include .env
export

migrate-up:
	go run cmd/migrator/main.go up

migrate-down: 
	go run cmd/migrator/main.go down

migrate-version:
	go run cmd/migrator/main.go version

migrate-steps:
	go run cmd/migrator/main.go steps $(n)

migrate-create:
	@echo "Enter migration name:" && read name && migrate create -ext sql -dir migrations -seq $$name

lint:
	golangci-lint -v run --config .golangci.yml

lint-fix:
	golangci-lint -v run --config .golangci.yml --fix

lint-fast:
	golangci-lint run --config .golangci.yml --fast
