ifneq ("$(wildcard .env)","")
	include .env
endif

MOCKERY_VERSION := v2.53.6
MOCKERY := bin/mockery

.PHONY: dev-env-start dev dev-client dev-start generate

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

generate: $(MOCKERY)
	$(MOCKERY)
	go generate ./...

$(MOCKERY):
	GOBIN=$(CURDIR)/bin go install github.com/vektra/mockery/v2@$(MOCKERY_VERSION)
