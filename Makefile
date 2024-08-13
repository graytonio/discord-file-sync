dev:
	docker compose up --build

build:
	goreleaser build --snapshot --single-target --clean