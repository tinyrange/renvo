FROM i386/debian:bookworm-slim@sha256:72b77a38b741753a64ebd7ff26c200d5eb5119f8b385409c5aaaa1aee12e9045

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends tcc libc6-dev binutils \
    && rm -rf /var/lib/apt/lists/*
