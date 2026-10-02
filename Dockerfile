FROM alpine

LABEL org.opencontainers.image.source=https://github.com/jamesread/StencilBox

RUN apk add --no-cache git npm

ENV PATH="/app/tools/node_modules/.bin:${PATH}"

COPY var/config-skel/ /config/
COPY var/tools/ /app/tools/
COPY templates/ /app/templates/
COPY layers/ /app/layers/
COPY StencilBox /app/StencilBox
COPY frontend/dist /frontend/

# OpenShift and other restricted platforms run as an arbitrary non-root UID; group 0
# must be able to write mounted or baked-in config and build output.
RUN mkdir -p /config/output \
	&& chgrp -R 0 /config \
	&& chmod -R g+rwX /config

WORKDIR /app

VOLUME /config

ENTRYPOINT ["/app/StencilBox"]
