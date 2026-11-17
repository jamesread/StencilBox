default: build

build:
	npm ci
	npx vite build --base '/webui/'

ci: build

test:
	@echo "No frontend unit tests yet"

lint:
	npm run lint

.PHONY: default build ci test lint
