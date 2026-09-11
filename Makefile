build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o tofu ./cmd/tofu

test:
	go test ./...

install:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o tofu ./cmd/tofu
	install -Dm755 tofu $(DESTDIR)/usr/local/bin/tofu
