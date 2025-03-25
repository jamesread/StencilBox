default:
	gomplate -d bookmarks=bookmarks.yaml -d links=links.yaml -f template.html > index.html

gomplate:
	wget https://github.com/hairyhenderson/gomplate/releases/download/v4.3.0/gomplate_linux-amd64
	mv gomplate_linux-amd64 gomplate
	chmod +x gomplate
	mv gomplate /usr/local/bin
