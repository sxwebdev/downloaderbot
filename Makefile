start:
	go run cmd/app/main.go

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
