#
# Build the Go server
#
FROM golang:1.12-alpine as go-build

WORKDIR /build

RUN apk update && apk add git gcc musl-dev

COPY cmd/ ./cmd
COPY pkg/ ./pkg
COPY go.mod ./
RUN ls -R

ENV GO111MODULE=on
WORKDIR /build/cmd/starter
# Disabling cgo results in a fully static binary that can run without C libs
RUN CGO_ENABLED=0 GOOS=linux go build

#
# Assemble the server binary and Vue bundle into a single app
#
FROM scratch
WORKDIR /app 

COPY --from=go-build /build/cmd/starter/starter ./starter

ENV PORT 8000
EXPOSE 8000
CMD ["/app/starter"]