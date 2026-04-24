FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY burrowd/go.mod burrowd/go.sum ./burrowd/
COPY protocol/go.mod ./protocol/

RUN go mod download -C burrowd

COPY burrowd/ ./burrowd/
COPY protocol/ ./protocol/

ARG TARGETARCH=amd64
WORKDIR /app/burrowd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /bin/burrowd .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
COPY --from=builder /bin/burrowd /usr/local/bin/
EXPOSE 25701
ENTRYPOINT ["burrowd"]
CMD ["serve"]
