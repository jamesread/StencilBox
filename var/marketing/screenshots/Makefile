# Regenerates marketing screenshots against a throwaway StencilBox seeded with
# fictional build configs. Run `make` here (or `make screenshots` from the repo
# root); everything is torn down afterwards.
#
# StencilBox uses SQLite next to config.yaml, so no database containers are needed.

ROOT := $(abspath ../../..)
HERE := $(abspath .)
TMP := $(HERE)/tmp

BASE_URL := http://127.0.0.1:18480
API := $(BASE_URL)/api/StencilBox.clientapi.v1.StencilBoxApiService

DOCS_ASSETS := $(ROOT)/docs/modules/ROOT/images

.PHONY: all build up seed shoot manifest publish down clean

all:
	@$(MAKE) build up seed shoot manifest publish; rc=$$?; $(MAKE) down; exit $$rc

build:
	$(MAKE) -C $(ROOT)/service build
	[ -d $(ROOT)/frontend/node_modules ] || (cd $(ROOT)/frontend && npm ci)
	$(MAKE) -C $(ROOT)/frontend build

up: down
	@if curl -fsS --max-time 2 $(BASE_URL)/webui/ >/dev/null 2>&1; then \
		echo "Port 18480 is already in use"; exit 1; fi
	rm -rf $(TMP)
	mkdir -p $(TMP)/output
	cp -a seed/. $(TMP)/
	# Started from service/ so webui resolves to ../frontend/dist and templates to ../templates.
	cd $(ROOT)/service && { \
		env PORT=18480 \
			STENCILBOX_DB_PATH=$(TMP)/stencilbox.db \
			STENCILBOX_OUTPUT_DIR=$(TMP)/output \
			STENCILBOX_SECURE_COOKIES=false \
			./StencilBox -configdir $(TMP) > $(TMP)/service.log 2>&1 & echo $$! > $(TMP)/service.pid; }
	# GetStatus is callable without login; Init requires authentication.
	@for i in $$(seq 60); do curl -fsS --max-time 2 -X POST -H 'Content-Type: application/json' -d '{}' \
		$(API)/GetStatus >/dev/null 2>&1 && exit 0; sleep 0.5; done; \
		echo "StencilBox did not start; see $(TMP)/service.log"; exit 1

seed:
	./seed.sh $(BASE_URL) $(TMP)

shoot:
	repo-helper screenshot --config screenshots.ini

manifest:
	./generate-manifest.py

publish:
	mkdir -p $(DOCS_ASSETS)
	cp build-configs.png $(DOCS_ASSETS)/ || true
	cp homepage.png $(DOCS_ASSETS)/homepage-built.png
	cp iframe-sidebar.png $(DOCS_ASSETS)/
	cp status-page.png $(DOCS_ASSETS)/

down:
	-@[ -f $(TMP)/service.pid ] && kill $$(cat $(TMP)/service.pid) 2>/dev/null; rm -f $(TMP)/service.pid

clean: down
	rm -rf $(TMP)
