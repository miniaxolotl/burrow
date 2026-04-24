FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY burrowd/go.mod ./burrowd/
COPY burrowd/go.sum ./burrowd/
COPY protocol/go.mod ./protocol/

RUN go mod download -C burrowd

WORKDIR /app
COPY burrowd/ ./burrowd/
COPY protocol/ ./protocol/

WORKDIR /app/burrowd
RUN go get burrow/protocol && go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /bin/burrowd .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/burrowd /usr/local/bin/

EXPOSE 25701 443

ENTRYPOINT ["burrowd"]
CMD ["serve"]
