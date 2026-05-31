FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/reward-system-users ./cmd/reward-system-users

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/reward-system-users /app/reward-system-users
COPY config/ /app/config/
COPY templates/ /app/templates/
COPY migrations/ /app/migrations/
ENV CONFIG_PATH=/app/config/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/reward-system-users"]
CMD ["serve"]
