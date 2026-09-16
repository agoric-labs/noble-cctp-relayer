FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS build-env

RUN apk add --update --no-cache curl make git libc-dev bash gcc linux-headers eudev-dev

ARG TARGETARCH
ARG BUILDARCH

RUN if [ "${TARGETARCH}" = "arm64" ] && [ "${BUILDARCH}" != "arm64" ]; then \
        wget -c https://musl.cc/aarch64-linux-musl-cross.tgz -O - | tar -xzvv --strip-components 1 -C /usr; \
    elif [ "${TARGETARCH}" = "amd64" ] && [ "${BUILDARCH}" != "amd64" ]; then \
        wget -c https://musl.cc/x86_64-linux-musl-cross.tgz -O - | tar -xzvv --strip-components 1 -C /usr; \
    fi

ADD . .

RUN if [ "${TARGETARCH}" = "arm64" ] && [ "${BUILDARCH}" != "arm64" ]; then \
        export CC=aarch64-linux-musl-gcc CXX=aarch64-linux-musl-g++;\
    elif [ "${TARGETARCH}" = "amd64" ] && [ "${BUILDARCH}" != "amd64" ]; then \
        export CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++; \
    fi; \
    GOOS=linux GOARCH=$TARGETARCH CGO_ENABLED=1 \
    LDFLAGS='-linkmode external -w -s -extldflags "-static"' \
    make install;

RUN if [ -d "/go/bin/linux_${TARGETARCH}" ]; then mv /go/bin/linux_${TARGETARCH}/* /go/bin/; fi

FROM alpine:3.20 AS rootfs
RUN apk add --no-cache ca-certificates \
 && addgroup -g 1000 -S strangelove \
 && adduser -u 100 -S -G strangelove strangelove \
 && mkdir -p /home/strangelove

# Use ln and rm from full featured busybox for assembling final image
FROM busybox:1.34.1-musl AS busybox-full

# Build final image from scratch
FROM scratch

LABEL org.opencontainers.image.source="https://github.com/strangelove-ventures/noble-cctp-relayer"

WORKDIR /bin

# Install ln (for making hard links) and rm (for cleanup) from full busybox image (will be deleted, only needed for image assembly)
COPY --from=busybox-full /bin/ln /bin/rm ./

COPY --from=busybox-full /bin/busybox /bin/sh

# Add hard links for read-only utils
# Will then only have one copy of the busybox minimal binary file with all utils pointing to the same underlying inode
RUN for b in \
  cat \
  date \
  df \
  du \
  env \
  grep \
  head \
  less \
  ls \
  md5sum \
  nc \
  nslookup \
  ping \
  ping6 \
  pwd \
  sha1sum \
  sha256sum \
  sha3sum \
  sha512sum \
  sleep \
  stty \
  tail \
  tar \
  tee \
  tr \
  watch \
  which \
  ; do ln sh $b; done

#  Remove write utils
RUN rm ln rm

# Install chain binaries
COPY --from=build-env /bin/noble-cctp-relayer /bin

COPY --from=rootfs /etc/ssl/certs/ca-certificates.crt /etc/ssl/cert.pem

# Install strangelove user
COPY --from=rootfs /etc/passwd /etc/passwd
COPY --from=rootfs --chown=100:1000 /home/strangelove /home/strangelove

WORKDIR /home/strangelove
USER strangelove

ENTRYPOINT ["/bin/noble-cctp-relayer"]
CMD ["start", "--config", "/source/config.yaml", "--log-format", "json"]