ARG ARCH=amd64
FROM ${ARCH}/golang:alpine AS build-env
COPY . $GOPATH/src/github.com/nickvanw/infping
WORKDIR $GOPATH/src/github.com/nickvanw/infping
RUN apk --update add git && go get -v && go build -o /infping

# final stage
FROM alpine
COPY --from=build-env /infping /
RUN apk add --no-cache ca-certificates fping
ENTRYPOINT ["/infping"]
