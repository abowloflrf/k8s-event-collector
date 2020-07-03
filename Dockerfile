FROM golang:1.14 as builder
WORKDIR /code
COPY go.mod go.sum /code/
RUN go version \
    && go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o k8s-events-dispatcher .

FROM alpine:3.12
COPY --from=builder /code/k8s-events-dispatcher /app
WORKDIR /app
ENTRYPOINT ["/app/k8s-events-dispatcher"]