FROM golang:1.26-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/user-service ./cmd/server

FROM alpine:3.23
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S -D -H -h /app -s /sbin/nologin -G app app \
    && mkdir -p /app \
    && chown -R app:app /app
WORKDIR /app
COPY --from=builder /out/user-service /usr/local/bin/user-service
RUN chown app:app /usr/local/bin/user-service
USER app:app
EXPOSE 8081
CMD ["user-service"]