FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared modules
COPY shared/ ./shared/

# Copy service code
COPY notification-service/ ./notification-service/

# Set working directory to service
WORKDIR /app/notification-service

# Download dependencies
RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/notification-service/notification-service .
CMD ["./notification-service"]

