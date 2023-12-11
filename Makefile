-include .env

docker_repo			= $(DOCKER_REPO)
docker_compose_cli	= docker compose -f docker-compose.yml -p downloaderbot

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

# Docker
docker-build:
	docker build -t ${docker_repo}:latest .

docker-push: docker-build
	docker push ${docker_repo}:latest
	docker image prune -f

# Infrasctructure
infra-start:
	$(docker_compose_cli) up -d $(filter-out $@,$(MAKECMDGOALS))

infra-stop:
	$(docker_compose_cli) stop $(filter-out $@,$(MAKECMDGOALS))

infra-update:
	$(docker_compose_cli) pull $(filter-out $@,$(MAKECMDGOALS))

infra-remove:
	$(docker_compose_cli) down

infra-logs:
	$(docker_compose_cli) logs -f $(filter-out $@,$(MAKECMDGOALS))

infra-exec:
	$(docker_compose_cli) exec $(filter-out $@,$(MAKECMDGOALS)) sh

web-minio:
	open http://localhost:${MINIO_CONSOLE_PORT}/
