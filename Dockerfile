# Build stage.
#
# No `go mod download` and no git: the dependency tree is private
# (github.com/nexsoft-git/* answers 401 to an anonymous git fetch, and
# proxy.golang.org does not carry it), so a container that tried to resolve it
# would need a git credential baked into the build. vendor/ is committed and
# carries the whole tree instead, which makes this stage hermetic - no network,
# no credentials, same result offline.
FROM golang:1.21-alpine AS build

WORKDIR /src

COPY . .

# -mod=vendor is the default once vendor/ exists; stated explicitly because it
# is the flag that means "this build touches no network", which is the whole
# point of the stage. CGO off so the binary is static - the runtime image
# carries no libc for it to link against.
RUN CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/sample-project .

# Runtime stage.
FROM alpine:3.19

# tzdata is not optional here: the schema stores `timestamp without time zone`,
# so the conversion CURRENT_TIMESTAMP performs on every write is a conversion
# into the session's TimeZone. Without the zone database the container can only
# ever be UTC, and TZ would be silently ignored.
# ca-certificates is for outbound TLS the service does not currently make.
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 app

WORKDIR /app

COPY --from=build /out/sample-project /app/sample-project

# i18n is read from ./i18n at startup - bundles.NewBundles walks the directory
# on disk - so it sits beside the binary rather than being embedded. A missing
# bundle directory is a startup failure, not a degraded mode.
COPY --chown=app:app i18n /app/i18n

# The swagger document is written to ./docs/swagger at startup. The controller
# creates that directory, but not the parent, so /app itself has to be
# writable by the runtime user.
RUN chown -R app:app /app

USER app

EXPOSE 8080

ENTRYPOINT ["/app/sample-project"]
