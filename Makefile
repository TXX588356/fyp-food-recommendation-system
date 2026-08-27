ifneq ("$(wildcard .env)","")
	include .env
endif

MOCKERY_VERSION := v3.5.5
MOCKERY := bin/mockery
MOCKERY_STAMP := bin/mockery-$(MOCKERY_VERSION)
OSTYPE ?= $(shell uname -s | tr A-Z a-z)
OSARCH ?= $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
GOTESTSUM := $(shell command -v gotestsum || echo "bin/gotestsum")
TPARSE := $(shell command -v tparse || echo "bin/tparse")
MINIO_CONTAINER ?= minio
POSTGRES_CONTAINER ?= fyp-postgres
CATALOG_DATA_DUMP ?= data/catalog-data.sql
CATALOG_DUMP_TABLES := meal_categories prebuilt_meals
MIGRATION_FILES := $(sort $(wildcard db/migrations/*.up.sql))
GO_CACHE_DIR ?= $(CURDIR)/out/go-build-cache
empty :=
space := $(empty) $(empty)
TEST_EXCLUDE_PACKAGES := \
	fyp/food-rs \
	fyp/food-rs/client/.* \
	fyp/food-rs/internal/mocks \
	fyp/food-rs/types/model \
	fyp/food-rs/internal/interfaces \
	fyp/food-rs/app/cmd \
	fyp/food-rs/app/cmd/server \

TEST_EXCLUDE_PATTERN := ^($(subst $(space),|,$(strip $(TEST_EXCLUDE_PACKAGES))))$$

.PHONY: dev-env-start dev dev-client dev-start generate dev-migrate migrate-up catalog-dump catalog-seed test test-cases

bin:
	mkdir -p bin

$(GOTESTSUM): VERSION := 1.11.0
$(GOTESTSUM): bin
	@if [ -f $@ ]; then \
		echo "gotestsum already installed, skipping download"; \
	else \
		echo "Installing gotestsum..."; \
		curl -Ls https://github.com/gotestyourself/gotestsum/releases/download/v$(VERSION)/gotestsum_$(VERSION)_$(OSTYPE)_$(OSARCH).tar.gz | tar -zOxf - gotestsum > $@ && chmod +x $@; \
	fi

$(TPARSE): VERSION := 0.13.2
$(TPARSE): bin
	@if [ -f $@ ]; then \
		echo "tparse already installed, skipping download"; \
	else \
		echo "Installing tparse..."; \
		curl -Ls https://github.com/mfridman/tparse/releases/download/v$(VERSION)/tparse_$(OSTYPE)_$(OSARCH) -o $@ && chmod +x $@; \
	fi

dev-env-start:
	@if podman container exists $(POSTGRES_CONTAINER); then \
		podman start $(POSTGRES_CONTAINER); \
	else \
		podman run -d \
			--name $(POSTGRES_CONTAINER) \
			-e POSTGRES_USER=${DATABASE_USER} \
			-e POSTGRES_PASSWORD=${DATABASE_PASSWORD} \
			-e POSTGRES_DB=${DATABASE_NAME} \
			-p ${DATABASE_PORT}:5432 \
			postgres; \
	fi
	@if podman container exists $(MINIO_CONTAINER); then \
		podman start $(MINIO_CONTAINER); \
	else \
		podman run -d \
			--name $(MINIO_CONTAINER) \
			-e MINIO_ROOT_USER=${OBJECT_STORAGE_ACCESS_KEY} \
			-e MINIO_ROOT_PASSWORD=${OBJECT_STORAGE_SECRET_KEY} \
			-p 9000:9000 \
			-p 9001:9001 \
			-v minio_data:/data \
			minio/minio server /data --console-address ":9001"; \
	fi
	@until curl -fsS http://${OBJECT_STORAGE_ENDPOINT}/minio/health/ready >/dev/null 2>&1; do \
		sleep 1; \
	done
	@podman run --rm --network host --entrypoint /bin/sh minio/mc -c \
		'mc alias set local http://${OBJECT_STORAGE_ENDPOINT} ${OBJECT_STORAGE_ACCESS_KEY} ${OBJECT_STORAGE_SECRET_KEY} >/dev/null && mc mb --ignore-existing local/${OBJECT_STORAGE_BUCKET}'

dev:
	go run main.go server

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

dev-migrate: dev-env-start migrate-up catalog-seed

# Apply every db/migrations/*.up.sql
migrate-up:
	@for migration in $(MIGRATION_FILES); do \
		echo "Applying $$migration"; \
		podman exec -i -e PGPASSWORD="${DATABASE_PASSWORD}" $(POSTGRES_CONTAINER) psql \
			-v ON_ERROR_STOP=1 \
			-U "${DATABASE_USER}" \
			-d "${DATABASE_NAME}" \
			< "$$migration"; \
	done

# Snapshot the catalog tables from current local Postgres database into a SQL file.
# Write to data/catalog-data.sql 
# export catalog tables from DB into data/catalog-data.sql
catalog-dump:
	@mkdir -p $(dir $(CATALOG_DATA_DUMP))
	podman exec -e PGPASSWORD="${DATABASE_PASSWORD}" $(POSTGRES_CONTAINER) pg_dump \
		-U "${DATABASE_USER}" \
		-d "${DATABASE_NAME}" \
		--data-only \
		--column-inserts \
		--no-owner \
		--no-privileges \
		$(foreach table,$(CATALOG_DUMP_TABLES),--table=$(table)) \
			> "$(CATALOG_DATA_DUMP)"
	@echo "Wrote catalog data dump to $(CATALOG_DATA_DUMP)"

# Loads data/catalog-data.sql into DB
catalog-seed:
	@test -f "$(CATALOG_DATA_DUMP)" || (echo "Missing $(CATALOG_DATA_DUMP). Run make catalog-dump first." && exit 1)
	podman exec -e PGPASSWORD="${DATABASE_PASSWORD}" $(POSTGRES_CONTAINER) psql \
		-v ON_ERROR_STOP=1 \
		-U "${DATABASE_USER}" \
		-d "${DATABASE_NAME}" \
		-c "DELETE FROM prebuilt_meals; DELETE FROM meal_categories;"
	podman exec -i -e PGPASSWORD="${DATABASE_PASSWORD}" $(POSTGRES_CONTAINER) psql \
		-v ON_ERROR_STOP=1 \
		-U "${DATABASE_USER}" \
		-d "${DATABASE_NAME}" \
		< "$(CATALOG_DATA_DUMP)"
	@echo "Seeded catalog data from $(CATALOG_DATA_DUMP)"

$(MOCKERY_STAMP):
	GOBIN=$(CURDIR)/bin go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)
	touch $(MOCKERY_STAMP)


out:
	mkdir -p out $(GO_CACHE_DIR)

test: export ENVIRONMENT := TEST
test: out $(GOTESTSUM) $(TPARSE)
	@packages="$$(GOCACHE="$(GO_CACHE_DIR)" go list ./... | grep -v -E '$(TEST_EXCLUDE_PATTERN)')" ; \
	coverage_file="out/coverage.out"; \
	test_json="out/gotestsum.json.out"; \
	test_junit="out/gotestsum.junit.xml"; \
	echo "==> Running Go tests"; \
	if $(GOTESTSUM) --format standard-verbose --jsonfile "$$test_json" --junitfile "$$test_junit" -- $$packages -vet=all -failfast -timeout=30s -test.coverprofile "$$coverage_file"; then \
		status=0; \
	else \
		status=$$?; \
	fi; \
	echo; \
	echo "==> tparse test result"; \
	$(TPARSE) -all -file "$$test_json"; \
	if [ -f "$$coverage_file" ]; then \
		echo; \
		echo "==> Overall coverage"; \
		GOCACHE="$(GO_CACHE_DIR)" go tool cover -func "$$coverage_file" | awk '/^total:/ { printf "TOTAL %-58s %s\n", $$2, $$3 }'; \
	fi; \
	exit $$status
