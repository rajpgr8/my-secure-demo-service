# Stage 1: Build
FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY app/ .
# Static compile is CRITICAL for distroless static images
RUN CGO_ENABLED=0 GOOS=linux go build -o myapp .

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /src/myapp /myapp
USER 65532:65532
ENTRYPOINT ["/myapp"]
