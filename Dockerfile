FROM golang:alpine as builder

WORKDIR /go/src/github.com/ETSGlobal/translations-refresher

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/refresher *.go


FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /go/src/github.com/ETSGlobal/translations-refresher/bin/refresher /usr/local/bin

ENTRYPOINT ["refresher"]
