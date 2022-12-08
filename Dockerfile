##
## Build
##
FROM golang:1.19-alpine3.17 AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./
RUN go mod download
RUN go build -o ./build/server ./pkg

##
## Deploy
##
FROM alpine:3.17
WORKDIR /

COPY --from=build /app/build/server /server
RUN ls -al
CMD ["/server"]
