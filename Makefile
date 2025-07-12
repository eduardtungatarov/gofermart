up:
	docker-compose up -d

run:
	go run cmd/gophermart/main.go

run-accrual:
	./cmd/accrual/accrual_darwin_amd64

migrate-create:
	goose -dir migrations create ${name} sql

fix:
	go vet ./... && go fmt ./... && goimports -w . && gofumpt -w -extra .

test:
	go test ./...

make gen-sql:
	sqlc generate

make push:
	git add . && git commit --amend -m "t" && git push origin -f iter1