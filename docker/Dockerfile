FROM scratch

ARG TARGETARCH

COPY ./bin/atkins-linux-${TARGETARCH} /usr/local/bin/atkins

WORKDIR /app

USER nobody

ENTRYPOINT ["/usr/local/bin/atkins"]
