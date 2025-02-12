FROM golang

WORKDIR /app
COPY . /app

RUN go build -o httpserver ./cmd/httpserver

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /app/httpserver .
CMD ["./httpserver"]
