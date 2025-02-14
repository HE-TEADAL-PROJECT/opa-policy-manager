FROM golang:1.23 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/main .

# Define environment variables
ENV MINIO_SERVER="" \
    MINIO_ACCESS_KEY="" \
    MINIO_SECRET_KEY="" \
    BUCKET_NAME="" \
    BUNDLE_NAME=""

EXPOSE 8080
CMD ["./main"]
