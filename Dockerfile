FROM golang:alpine as builder

WORKDIR /build

COPY . . 
RUN go mod download

RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -o /bin/main cmd/labns/main.go

FROM alpine:3.13

# PATH TO MOUNT CONFIG FILE AT RUNTIME
ENV LABNS_CONFIG_PATH=/dist/config.json

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
WORKDIR /dist
COPY --from=builder /bin/main .

EXPOSE 53

CMD ["/dist/main"]