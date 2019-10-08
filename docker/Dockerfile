ARG ARCH=amd64

FROM golang:alpine AS build-env
ARG ARCH
COPY . $GOPATH/src/github.com/majorcadevs/infping
WORKDIR $GOPATH/src/github.com/majorcadevs/infping
RUN apk add git --no-cache && \
    case ${ARCH} in \
        amd64) export GOARCH=amd64 ;; \
        arm32v7) export GOARCH=arm; export GOARM=7 ;; \
        arm64v8) export GOARCH=arm64 ;; \
        *) echo Unsupported arch ${ARCH}; exit 1 ;; \
    esac && \
    go get -v && \
    go build -o /infping -v

# final stage
FROM ${ARCH}/alpine
COPY --from=build-env /infping /
RUN apk add --no-cache ca-certificates fping
ENTRYPOINT ["/infping"]
