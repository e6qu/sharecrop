# syntax=docker/dockerfile:1

# Container image for the sharecrop backend: the same wasm app that runs in the
# browser demo, hosted server-side through the WASI guest pool. Everything is
# baked in at build time - the frontend (embedded in the binary), the wasip1
# guest, and the guest's compiled machine code (a wazero AOT cache) - so the
# server does NO build on startup.
#
# The build is native to the target platform (no cross-compilation): the guest
# is compiled into the wazero cache by running the freshly built binary, which
# must execute on the target CPU, and the cache is CPU-arch-specific. Build each
# arch on a native runner (CI does this in an arm64/amd64 matrix); building the
# non-native arch locally works but runs the whole build under emulation. See
# docs/deployment.md for the image-naming standard and the ECS Fargate setup.
FROM golang:1.26-bookworm AS build
WORKDIR /src

# Resolve modules first so the dependency layer caches across source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# The committed guest at internal/wasiguest/app-guest.wasm is an empty
# placeholder; build the real wasip1 guest so the host binary embeds it. The
# guest is wasm (architecture-independent).
RUN GOOS=wasip1 GOARCH=wasm go build -trimpath -o internal/wasiguest/app-guest.wasm ./cmd/sharecrop-wasi-app-guest

# Build a static (CGO-free) binary for the build platform's arch. It embeds the
# guest and the committed frontend (web/static), so the runtime image needs
# nothing but this binary, the wazero cache, and the migrations directory.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sharecrop ./cmd/sharecrop

# Pre-compile the guest into a wazero cache so serve loads machine code instead
# of compiling the ~11 MB module on boot (~1.7s cold -> ~0.07s warm). This runs
# the binary just built, so the cache matches its exact wazero version and CPU.
RUN /out/sharecrop wasi-precompile /wazero-cache

# Minimal, non-root runtime. distroless/static carries only CA certificates
# (needed for TLS to Postgres/RDS) and tzdata, and is selected per target arch
# automatically. There is no shell or curl: the binary's `healthcheck` command
# probes the running server for the Amazon ECS container health check.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/sharecrop /usr/local/bin/sharecrop
# The cache is written by root in the build stage; the runtime runs as the
# distroless nonroot user (uid 65532), which must be able to read it.
COPY --from=build --chown=65532:65532 /wazero-cache /wazero-cache
COPY --from=build /src/migrations /migrations
ENV SHARECROP_MIGRATIONS_DIR=/migrations \
    SHARECROP_HTTP_ADDR=:8080 \
    SHARECROP_WAZERO_CACHE_DIR=/wazero-cache
EXPOSE 8080
# `serve` runs the stateless backend; override with `migrate up` for the one-off
# migration task. DATABASE_URL and SHARECROP_ACCESS_TOKEN_SECRET come from the
# task definition (secrets), never baked into the image.
ENTRYPOINT ["/usr/local/bin/sharecrop"]
CMD ["serve"]
