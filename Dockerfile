##
## Build
##
FROM golang:1.21-alpine3.17 AS build

WORKDIR /app

RUN apk add build-base

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go install github.com/google/wire/cmd/wire@latest

COPY . ./
RUN wire ./pkg
RUN go build -o ./build/server ./pkg

##
## Deploy
##
FROM alpine:3.17
WORKDIR /

COPY --from=build /app/build/server /server
RUN ls -al
ENTRYPOINT ["/server"]
