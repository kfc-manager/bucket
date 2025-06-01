FROM golang:1.24.1-alpine3.21 AS build

WORKDIR /app

COPY go.mod ./

COPY main.go ./main.go
COPY domain ./domain
COPY server ./server

RUN go build -o bin main.go

FROM alpine:3.21

COPY --from=build /app/bin /main
RUN mkdir data

ENTRYPOINT [ "/main" ]
