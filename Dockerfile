FROM golang:alpine as build
ADD main.go /go/src/github.com/tjamet/httar/main.go
RUN go build -o /bin/httar /go/src/github.com/tjamet/httar/main.go

FROM alpine
COPY --from=build /bin/httar /bin/httar
ENTRYPOINT ["/bin/httar"]
