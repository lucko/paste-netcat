FROM golang:alpine as build

# copy source files
WORKDIR $GOPATH/src/github.com/lucko/paste-netcat/
COPY go.mod .
COPY main.go .

# build app
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /go/bin/paste-netcat

# create minimal production image
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/bin/paste-netcat /usr/bin/
ENTRYPOINT ["paste-netcat"]
