# Copyright 2025 Matteo Brambilla - TEADAL
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.23.6-alpine

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY /testdata/ ./testdata/
COPY /static/ ./static/
COPY /cmd/web/ ./cmd/web/
COPY /internal/ ./internal/
RUN go build -o main ./cmd/web

# Define environment variables
# ENV MINIO_SERVER="" \
#     MINIO_ACCESS_KEY="" \
#     MINIO_SECRET_KEY="" \
#     BUCKET_NAME="" \
#     BUNDLE_NAME=""

# EXPOSE 8080
ENTRYPOINT ["./main"]
