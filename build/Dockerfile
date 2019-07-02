# =========================================================================================
# === Build go app ========================================================================
# =========================================================================================
FROM golang:1.12-alpine as go-build
WORKDIR /build
ARG goPackage="github.com/benc-uk/go-starter/cmd/starter"
ARG version="0.0.1"
ARG buildInfo="Local Docker build"

ENV PORT 8000

# Install system dependencies
RUN apk update && apk add git gcc musl-dev

# Fetch and cache Go modules
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy in Go source files
COPY cmd/ ./cmd
COPY pkg/ ./pkg

# Now run the build
# Disabling cgo results in a fully static binary that can run without C libs
# Also inject version and build details 
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=$version -X 'main.buildInfo=$buildInfo'" \
    -o server \
    $goPackage

# =========================================================================================
# === Create production/runtime image =====================================================
# =========================================================================================
FROM scratch
WORKDIR /app 
EXPOSE 8000
COPY --from=go-build /build/server . 
CMD [ "./server" ]