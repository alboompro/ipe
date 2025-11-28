# Build stage
FROM golang:alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ipe ./cmd

# Runner Stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/ipe /bin/ipe

RUN chmod +x /bin/ipe

EXPOSE 8080
EXPOSE 4433

CMD ["/bin/ipe", "--config", "/config/config.yml"]
