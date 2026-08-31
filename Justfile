# Everything goes through here; CI runs the same recipes.

# Build and vet.
check:
    go vet ./...
    go test ./...
    go build ./...

# Run against a HomeBox instance over stdio (the default transport).
run url token:
    HOMEBOX_URL={{url}} HOMEBOX_TOKEN={{token}} go run ./cmd/homebox-mcp

# Serve over HTTP, as it runs in a cluster. NOTE: no authentication --
# put a gateway in front of it.
serve url token addr="127.0.0.1:8080":
    HOMEBOX_URL={{url}} HOMEBOX_TOKEN={{token}} TRANSPORT=http ADDR={{addr}} \
        go run ./cmd/homebox-mcp

# What the release will build, without publishing.
snapshot:
    goreleaser release --snapshot --clean --skip=publish
