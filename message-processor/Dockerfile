FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared modules
COPY shared/ ./shared/

# Copy service code
COPY message-processor/ ./message-processor/

# Set working directory to service
WORKDIR /app/message-processor

# Download dependencies
RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o message-processor .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/message-processor/message-processor .
CMD ["./message-processor"]
