FROM alpine

LABEL org.opencontainers.image.source=https://github.com/jamesread/StencilBox

#COPY config.yaml /config/config.yaml
COPY StencilBox /app/StencilBox
#COPY frontend/dist /webui

VOLUME /config

ENTRYPOINT ["/app/StencilBox"]
