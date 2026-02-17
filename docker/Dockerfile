FROM alpine:latest

ARG TARGETARCH

# Install bash (required for command execution)
RUN apk add --no-cache bash

# Verify that uname exists, used for tests
RUN which uname

# Create the nobody user if it does not already exist
RUN id nobody 2>/dev/null || adduser -D -H -u 65534 nobody

COPY ./bin/atkins-linux-${TARGETARCH} /usr/local/bin/atkins

WORKDIR /app

ENTRYPOINT ["/usr/local/bin/atkins"]
