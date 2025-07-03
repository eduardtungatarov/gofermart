migrate-create:
	goose -dir migrations create ${name} sql

up:
	docker-compose up -d

run:
	go run cmd/gophermart/main.go

fix:
	go vet ./... && go fmt ./... && goimports -w . && gofumpt -w -extra .