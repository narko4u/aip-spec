# ── AIP — Agent Interaction Protocol CLI ───────────────────
# https://github.com/narko4u/aip-spec
# ghcr.io/narko4u/aip-spec

FROM alpine:3.20

ARG VERSION=0.2.0
ARG TARGETARCH=amd64

RUN apk add --no-cache ca-certificates wget && \
    wget -q "https://github.com/narko4u/aip-spec/releases/download/v${VERSION}/aip_${VERSION}_linux_${TARGETARCH}.tar.gz" -O - | \
    tar xz -C /usr/local/bin/ aip && \
    apk del wget

ENTRYPOINT ["aip"]
CMD ["--help"]
