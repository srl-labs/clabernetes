.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

regenerate-types: ## Regenerate types from the clabernetes-openapi.json file in this dir.
	npx @hey-api/openapi-ts --useOptions
