FROM golang:alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /pinger ./cmd/pinger

FROM gcr.io/distroless/static
COPY --from=builder /pinger /pinger
EXPOSE 8080
ENTRYPOINT ["/pinger"]
