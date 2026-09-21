# --- build stage ---
FROM golang:1.22-bookworm AS build
WORKDIR /src

# NOTE: this repo's go.mod contains `replace` directives routing
# golang.org/x/*, gopkg.in/*, and a couple of other vanity-import domains
# through their identical github.com mirrors. This was required in the
# sandboxed environment this project was built in, which allowlisted
# github.com but not golang.org/gopkg.in for outbound network access. If
# your Docker build environment has normal unrestricted internet access,
# these replace directives are harmless (they point at byte-identical
# mirrors) and can be left in place or removed -- see go.mod's header
# comment for details.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# --- runtime stage ---
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=build /out/server ./server
COPY migrations ./migrations

RUN mkdir -p /app/uploads/resumes && \
    addgroup -S app && adduser -S app -G app && \
    chown -R app:app /app
USER app

EXPOSE 8080
ENTRYPOINT ["./server"]
