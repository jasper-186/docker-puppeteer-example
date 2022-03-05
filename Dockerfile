# syntax=docker/dockerfile:1

FROM golang:alpine AS build

RUN apk update

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -o ./server.out

##
## Deploy
##
FROM alpine
RUN apk update
WORKDIR /

COPY --from=build /app/server.out /server.out

EXPOSE 8080

RUN addgroup -S nonroot && adduser -S nonroot -G nonroot 
USER nonroot

ENTRYPOINT ["/server.out"]