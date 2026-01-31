.PHONY: migrate-up migrate-down migrate-version migrate-steps migrate-create

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