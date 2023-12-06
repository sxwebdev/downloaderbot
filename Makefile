start:
	go run cmd/app/main.go

build:
	go build -o ./.build/app -v cmd/app/main.go

watch:
	air -c .air.toml

markdown:
	go run -v ./cmd/app --markdown --file ENVS.md

grpcui:
	grpcui --plaintext localhost:9000

genproto:
	rm -rf pb/*
	protoc \
	--go_out=:pb \
	--go-grpc_out=:pb \
	proto/*.proto
