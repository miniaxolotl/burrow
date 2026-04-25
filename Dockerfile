FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY burrowd/go.mod burrowd/go.sum ./burrowd/
COPY protocol/go.mod ./protocol/

RUN go mod download -C burrowd

COPY burrowd/ ./burrowd/
COPY protocol/ ./protocol/

ARG TARGETARCH
ARG VERSION=dev
WORKDIR /app/burrowd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-$(go env GOARCH)} \
    go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /bin/burrowd .

FROM alpine:3.20

LABEL org.opencontainers.image.source="https://github.com/miniaxolotl/burrow" \
      org.opencontainers.image.url="https://github.com/miniaxolotl/burrow" \
      org.opencontainers.image.documentation="https://github.com/miniaxolotl/burrow#readme" \
      org.opencontainers.image.title="burrowd" \
      org.opencontainers.image.description="Self-hosted tunnel server" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="Elias Mawa" \
      org.opencontainers.image.authors="Elias Mawa <elias@mawa.dev>"

RUN apk add --no-cache ca-certificates tzdata wget
COPY --from=builder /bin/burrowd /usr/local/bin/
EXPOSE 25701
ENTRYPOINT ["burrowd"]
CMD ["serve"]
