-include .env

docker_repo					= $(DOCKER_REPO)
docker_compose_cli	= docker compose -f docker-compose.yml -p downloaderbot

start:
	go run ./cmd/app start

build:
	go build -o ./.build/app -v cmd/app/main.go

test:
	go test -race -count=1 ./...

test-cover:
	go test -race -count=1 -coverprofile=cover.out ./... && go tool cover -html=cover.out

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

watch:
	air -c .air.toml

fmt:
	gofumpt -l -w .

grpcui:
	grpcui --plaintext localhost:9000

genenvs:
	go run ./cmd/app config genenvs

genproto:
	rm -rf pb/*
	protoc \
	--go_out=:pb \
	--go-grpc_out=:pb \
	proto/*.proto

# Docker
docker-push:
	docker buildx build --platform linux/amd64 --push \
		--build-arg VERSION=`git describe --tags --abbrev=0 || echo "0.0.0"` \
		--build-arg COMMIT_HASH=`git rev-parse --short HEAD` \
		--build-arg BUILD_DATE=`date -u +'%Y-%m-%dT%H:%M:%SZ'` \
		-t ${docker_repo}:latest .

# Local image build for testing (loads into the local docker daemon, no push)
docker-build:
	docker build \
		--build-arg VERSION=`git describe --tags --abbrev=0 || echo "0.0.0"` \
		--build-arg COMMIT_HASH=`git rev-parse --short HEAD` \
		--build-arg BUILD_DATE=`date -u +'%Y-%m-%dT%H:%M:%SZ'` \
		-t downloaderbot:test .

# Run the locally built test image (requires .env with TELEGRAM_BOT_API_TOKEN)
docker-run: docker-build
	docker run --rm -it --env-file .env downloaderbot:test start

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

# Release
release:
	@if [ -z "$(TAG)" ]; then echo "Usage: make release TAG=v1.2.3"; exit 1; fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)

# Local dry run of the release: builds binaries and renders the changelog
# into ./dist without creating a tag or publishing anything.
release-preview:
	goreleaser release --snapshot --clean --skip=publish
