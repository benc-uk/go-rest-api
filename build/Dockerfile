# ===================================================================================
# === Stage 1: Build the Go backend service =========================================
# ===================================================================================
FROM golang:1.21-alpine as go-build
WORKDIR /build

ARG GO_PKG="github.com/benc-uk/go-rest-api/cmd"
ARG VERSION="0.0.1"
ARG BUILD_INFO="Local Docker build"
ARG CGO_ENABLED=0

# Install system dependencies, if CGO_ENABLED=1
RUN if [[ $CGO_ENABLED -eq 1 ]]; then apk update && apk add gcc musl-dev; fi

# Fetch and cache Go modules
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy in Go source files
COPY cmd/ ./cmd
COPY pkg/ ./pkg

# Now run the build
# Inject version and build details, to be available at runtime 
RUN GO111MODULE=on CGO_ENABLED=$CGO_ENABLED GOOS=linux \
  go build \
  -ldflags "-X main.version=$VERSION -X 'main.buildInfo=$BUILD_INFO'" \
  -o server ${GO_PKG}

# ================================================================================================
# === Stage 2: Get backend binary into a lightweight container ===================================
# ================================================================================================
FROM scratch
WORKDIR /app 

ENV PORT=8000

EXPOSE 8000

COPY --from=go-build /build/server . 
CMD [ "./server" ]