FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/migrator ./cmd/migrator

FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/api /app/api
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM alpine:3.21 AS migrator
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/migrator /app/migrator
COPY migrations /app/migrations
ENV MIGRATIONS_PATH=file:///app/migrations
USER appuser
ENTRYPOINT ["/app/migrator"]
CMD ["up"]
