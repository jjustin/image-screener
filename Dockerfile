FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o image-screener .

FROM scratch
COPY --from=builder /app/image-screener /image-screener
COPY static/ /static/
EXPOSE 8080
ENTRYPOINT ["/image-screener"]
CMD ["-addr=:8080"]
