# syntax=docker/dockerfile:1

# Container image for the sharecrop backend: the same wasm app that runs in the
# browser demo, hosted server-side through the WASI guest pool. The build runs on
# the native build platform (no QEMU emulation) and only the final host binary is
# cross-compiled to the target arch, so multi-arch builds stay fast. See
# docs/deployment.md for the image-naming standard and the ECS Fargate setup.
FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS build
WORKDIR /src

# Resolve modules first so the dependency layer caches across source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# The committed guest at internal/wasiguest/app-guest.wasm is an empty
# placeholder; build the real wasip1 guest so the host binary embeds it and
# serves dynamic routes through the WASI runtime instead of falling back to the
# native mux. The guest is wasm (architecture-independent), so it is built once
# regardless of the target arch.
RUN GOOS=wasip1 GOARCH=wasm go build -o internal/wasiguest/app-guest.wasm ./cmd/sharecrop-wasi-app-guest

# Cross-compile a static (CGO-free) linux binary for the requested architecture.
# It embeds the guest above and the committed frontend assets (web/static), so
# the runtime image needs nothing but this binary and the migrations directory.
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/sharecrop ./cmd/sharecrop

# Minimal, non-root runtime. distroless/static carries only CA certificates and
# tzdata and is selected per target arch automatically, so the image is small
# and fast to pull. There is no shell or curl: liveness/readiness is an external
# ALB health check against /healthz (see docs/deployment.md).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/sharecrop /usr/local/bin/sharecrop
COPY --from=build /src/migrations /migrations
ENV SHARECROP_MIGRATIONS_DIR=/migrations \
    SHARECROP_HTTP_ADDR=:8080
EXPOSE 8080
# `serve` runs the stateless backend; override with `migrate up` for the one-off
# migration task. DATABASE_URL and SHARECROP_ACCESS_TOKEN_SECRET come from the
# task definition (secrets), never baked into the image.
ENTRYPOINT ["/usr/local/bin/sharecrop"]
CMD ["serve"]
