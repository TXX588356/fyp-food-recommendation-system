ifneq ("$(wildcard .env)","")
	include .env
endif

dev-env-start:
	podman start fyp-postgres

dev:
	go run main.go

dev-client:
	pnpm -C client dev

dev-start:
	podman run -d \
  	--name fyp-postgres \
  	-e POSTGRES_USER=${DATABASE_USER} \
  	-e POSTGRES_PASSWORD=${DATABASE_PASSWORD} \
  	-e POSTGRES_DB=${DATABASE_NAME} \
  	-p ${DATABASE_PORT}:5432 \
  	postgres

