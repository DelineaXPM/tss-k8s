FROM alpine:latest AS build
RUN apk update && apk upgrade
RUN apk add go

WORKDIR /b
COPY cmd/ ./cmd
COPY pkg/ ./pkg
COPY go.mod go.sum ./
RUN go build cmd/tss-injector-svc.go

FROM alpine:latest
RUN apk update && apk upgrade
RUN addgroup tss && adduser -S -G tss tss

COPY --from=build /b/tss-injector-svc /usr/bin/

USER tss
WORKDIR /home/tss
