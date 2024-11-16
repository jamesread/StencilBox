default:
	gomplate -d links=links.yaml -f sidebar.template.html > sidebar.output.html
