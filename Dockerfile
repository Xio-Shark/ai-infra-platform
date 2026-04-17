FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api-server ./cmd/api-server
RUN CGO_ENABLED=0 go build -o /bin/gateway ./cmd/gateway
RUN CGO_ENABLED=0 go build -o /bin/notifier ./cmd/notifier

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /bin/api-server /bin/api-server
COPY --from=builder /bin/gateway /bin/gateway
COPY --from=builder /bin/notifier /bin/notifier
EXPOSE 8080
ENTRYPOINT ["/bin/api-server"]
