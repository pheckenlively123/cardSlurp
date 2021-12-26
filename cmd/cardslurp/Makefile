TARGET=$(GOPATH)/bin

mac_m1: clean clean_win
	env GOOS=darwin GOARCH=arm64 go build -o cardslurp main.go

mac_intel: clean clean_win
	env GOOS=darwin GOARCH=amd64 go build -o cardslurp main.go

linux_intel: clean clean_win
	env GOOS=linux GOARCH=amd64 go build -o cardslurp main.go

win_intel: clean clean_win
	env GOOS=windows GOARCH=amd64 go build -o cardslurp.exe main.go

clean:
	rm -f cardslurp

clean_win:
	rm -f cardslurp.exe

install:
	mkdir -p $(TARGET)
	cp cardSlurp $(TARGET)
