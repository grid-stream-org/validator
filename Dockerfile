FROM golang:1.23-alpine AS builder
RUN apk add --no-cache
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o build/validator ./cmd/validator

FROM alpine:latest
RUN apk add --no-cache ca-certificates && \
    adduser -D nonroot
WORKDIR /app
COPY --from=builder /src/build/validator .
USER nonroot
CMD ["/app/validator"]