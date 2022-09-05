mac_m1: clean
	GOOS=darwin GOARCH=arm64 go build

mac_intel: clean
	GOOS=darwin GOARCH=amd64 go build

win_intel: clean
	GOOS=windows GOARCH=amd64 go build
	mv cardslurp.exe cardslurp-new.exe
	cp cardslurp-new.exe /home/pheckenl/Pics/RawPics-Work-Queue/2022-09-04-DeleteMe

linux_intel: clean
	GOOS=linux GOARCH=amd64 go build

all: cardslurp

clean:
	rm -f cardslurp

install: all
	mkdir -p $(TARGET)
	cp cardslurp $(TARGET)
