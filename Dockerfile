FROM golang:1.23.1-alpine

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .
RUN go build -o main .

# Define environment variables
ENV MINIO_SERVER="" \
    MINIO_ACCESS_KEY="" \
    MINIO_SECRET_KEY="" \
    BUCKET_NAME="" \
    BUNDLE_NAME=""

EXPOSE 8080
CMD ["./main"]
