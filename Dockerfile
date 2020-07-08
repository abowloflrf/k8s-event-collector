FROM golang:1.14 as builder
WORKDIR /code
COPY go.mod go.sum /code/
RUN go version \
    && go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o event-collector .

FROM alpine:3.12
WORKDIR /app
COPY --from=builder /code/event-collector /app
ENTRYPOINT ["/app/event-collector"]