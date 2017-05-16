FROM golang:1.7-alpine

RUN apk add --no-cache bash git openssh && go get github.com/garyburd/redigo/redis
ADD . /go

