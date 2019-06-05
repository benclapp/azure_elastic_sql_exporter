FROM alpine:latest as certificates
RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates


FROM golang:1.12 as builder

WORKDIR /go/src/github.com/benclapp/azure_elastic_sql_exporter
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(cat VERSION)" -o /app/azure_elastic_sql_exporter

FROM scratch

# Copy certs from alpine as they don't exist from scratch
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/azure_elastic_sql_exporter /app/

WORKDIR /app
ENTRYPOINT ["/app/azure_elastic_sql_exporter"]
