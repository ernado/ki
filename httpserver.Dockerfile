FROM golang:1.23-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o httpserver ./cmd/httpserver

FROM alpine:edge

WORKDIR /app
COPY --from=build /app/httpserver .
RUN apk --no-cache add ca-certificates tzdata

ENTRYPOINT ["/app/httpserver"]
