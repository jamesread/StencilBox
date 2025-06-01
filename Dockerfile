FROM alpine

LABEL org.opencontainers.image.source=https://github.com/jamesread/StencilBox

COPY var/config-skel/ /config/
COPY templates/ /app/templates/
COPY StencilBox /app/StencilBox
COPY frontend/dist /frontend/

VOLUME /config

ENTRYPOINT ["/app/StencilBox"]
