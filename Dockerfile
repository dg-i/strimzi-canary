FROM scratch

# set automatically by BuildKit / buildx, the binary for each platform is built by "make go_build" with the corresponding GOARCH
ARG TARGETOS
ARG TARGETARCH

ADD cmd/target/${TARGETOS}/${TARGETARCH}/strimzi-canary ./

LABEL org.opencontainers.image.source='https://github.com/dg-i/strimzi-canary'

ENTRYPOINT ["/strimzi-canary"]
