GOCMD=go
GOBLD=$(GOCMD) build
TARGET=$(GOPATH)/bin

cardSlurp: cardSlurp.go card_file_util/*.go file_control/*.go
	$(GOBLD) cardSlurp.go

all: cardSlurp

clean:
	rm cardSlurp

install: all
	mkdir -p $(TARGET)
	cp cardSlurp $(TARGET)
