FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /app

COPY burrowd/go.mod burrowd/go.sum ./burrowd/
COPY protocol/go.mod ./protocol/

RUN go -C burrowd mod download

COPY burrowd/ ./burrowd/
COPY protocol/ ./protocol/

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /app/burrowd
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-$(go env GOARCH)} \
    go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /bin/burrowd .

FROM alpine:3.23

LABEL org.opencontainers.image.source="https://github.com/miniaxolotl/burrow" \
      org.opencontainers.image.url="https://useburrow.app" \
      org.opencontainers.image.documentation="https://github.com/miniaxolotl/burrow#readme" \
      org.opencontainers.image.title="burrowd" \
      org.opencontainers.image.description="Self-hosted tunnel server" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="Bramble Labs" \
      org.opencontainers.image.authors="Elias Mawa <elias@mawa.dev>"

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S burrow && adduser -S -G burrow burrow
COPY --from=builder /bin/burrowd /usr/local/bin/
USER burrow
EXPOSE 25701
ENTRYPOINT ["burrowd"]
CMD ["serve"]
