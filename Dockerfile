FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api-server ./cmd/api-server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /bin/api-server /bin/api-server
EXPOSE 8080
ENTRYPOINT ["/bin/api-server"]
