FROM debian/eol:jessie@sha256:81cc60273fcd009665b917085e8edb4a44af01130e42a9349545bcaaf05236a2

RUN apt-get -o Acquire::Check-Valid-Until=false update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends gcc-multilib binutils \
    && rm -rf /var/lib/apt/lists/*
