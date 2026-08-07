default: protocol frontend service

local:
	gomplate -d bookmarks=bookmarks.yaml -d links=links.yaml -f template.html > index.html

service: container-tools
	$(MAKE) -wC service

frontend:
	$(MAKE) -wC frontend

protocol:
	$(MAKE) -wC protocol

test:
	$(MAKE) -wC service test
	$(MAKE) -wC frontend test

lint:
	$(MAKE) -wC service lint
	$(MAKE) -wC frontend lint

integration-test:
	$(MAKE) -wC integration-tests

docs:
	$(MAKE) -wC docs
	./docs/node_modules/.bin/antora antora-playbook.yml

container-tools:
	$(MAKE) -wC var/tools/

gomplate:
	wget https://github.com/hairyhenderson/gomplate/releases/download/v4.3.0/gomplate_linux-amd64
	mv gomplate_linux-amd64 gomplate
	chmod +x gomplate
	mv gomplate /usr/local/bin


.PHONY: default service frontend docs gomplate protocol test lint integration-test
