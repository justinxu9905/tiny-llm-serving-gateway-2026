PROTO_SRC  := proto/gateway/v1/gateway.proto
PROTOC_FLAGS := -I proto \
	--go_out=gen --go_opt=paths=source_relative \
	--go-grpc_out=gen --go-grpc_opt=paths=source_relative

.PHONY: proto build run tidy bazel-build bazel-run bazel-test gazelle

proto:
	@mkdir -p gen
	PATH=$(PATH):$(HOME)/go/bin protoc $(PROTOC_FLAGS) $(PROTO_SRC)

build:
	go build ./cmd/server

run:
	go run ./cmd/server

tidy:
	go mod tidy

# ── Bazel targets ─────────────────────────────────────────────────────────────

bazel-build:
	bazel build //cmd/server

bazel-run:
	bazel run //cmd/server

bazel-test:
	bazel test //...

# Regenerate BUILD.bazel files from Go sources.
# Run this after adding/removing files or changing imports.
gazelle:
	bazel run //:gazelle
