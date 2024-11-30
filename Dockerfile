FROM golang:1.22.0-alpine
RUN apk add --no-cache gcc musl-dev bash curl
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.52.2

WORKDIR /app
COPY . .

CMD ["bash", "./scripts/run_tests.sh"]
