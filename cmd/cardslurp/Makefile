mac_arm64: clean lint scan test
	GOOS=darwin GOARCH=arm64 go build

mac_amd64: clean lint scan test
	GOOS=darwin GOARCH=amd64 go build

win_amd64: clean lint scan test
	GOOS=windows GOARCH=amd64 go build

linux_amd64: clean lint scan test
	GOOS=linux GOARCH=amd64 go build

test:
	cd ../../internal/cardfileutil && go test && cd ../../cmd/cardslurp
	cd internal/filecontrol && go test && cd ../..

# See this page for install instructions for golanci-lint: https://golangci-lint.run/usage/install/
lint:
	golangci-lint run ./...

# See this page for install instructions for govulncheck: https://go.dev/blog/vuln
scan:
	cd ../.. && govulncheck ./... && cd cmd/cardslurp

all: cardslurp

clean:
	rm -f cardslurp cardslurp.exe
