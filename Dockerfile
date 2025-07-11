FROM alpine

LABEL org.opencontainers.image.source=https://github.com/jamesread/StencilBox

RUN apk add --no-cache git

COPY var/config-skel/ /config/
COPY var/tools/ /app/tools/
COPY templates/ /app/templates/
COPY layers/ /app/layers/
COPY StencilBox /app/StencilBox
COPY frontend/dist /frontend/

VOLUME /config

ENTRYPOINT ["/app/StencilBox"]
