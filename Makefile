.PHONY: deploy

deploy:
	@if [ ! -f .env ]; then cp .env.example .env; chmod 600 .env; echo 'Created .env; review its credentials before exposing the service publicly.'; fi
	@data_path=$$(./scripts/resolve-data-path.sh); \
	builder_cache_path=$${DOCKER_BUILDER_CACHE_PATH:-$$(dirname "$$data_path")/docker-build-cache}; \
	mkdir -p "$$data_path/tmp" "$$builder_cache_path"; \
	chmod 0750 "$$data_path" "$$builder_cache_path"; \
	chmod 1777 "$$data_path/tmp"; \
	DOCKER_PB_DATA_PATH="$$data_path" DOCKER_BUILDER_CACHE_PATH="$$builder_cache_path" \
	docker compose -p ebook-reader-uzqw -f compose.yaml -f compose.build.yaml up -d --build
