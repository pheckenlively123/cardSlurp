mac_arm: clean
	GOOS=darwin GOARCH=arm64 go build

mac_intel: clean
	GOOS=darwin GOARCH=amd64 go build

win_intel: clean
	GOOS=windows GOARCH=amd64 go build

linux_intel: clean
	GOOS=linux GOARCH=amd64 go build

all: cardslurp

clean:
	rm -f cardslurp cardslurp.exe
