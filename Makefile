ifneq ("$(wildcard .env)","")
	include .env
endif

MOCKERY_VERSION := v3.5.5
MOCKERY := bin/mockery
MOCKERY_STAMP := bin/mockery-$(MOCKERY_VERSION)

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

generate: $(MOCKERY_STAMP)
	$(MOCKERY)
	go generate ./...

$(MOCKERY_STAMP):
	GOBIN=$(CURDIR)/bin go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)
	touch $(MOCKERY_STAMP)
