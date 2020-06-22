GOCMD=go
GOBLD=$(GOCMD) build
TARGET=$(GOPATH)/bin

cardSlurp: cardSlurp.go cardFileUtil/*.go fileControl/*.go
	$(GOBLD) cardSlurp.go

all: cardSlurp

clean:
	rm cardSlurp

install: all
	mkdir -p $(TARGET)
	cp cardSlurp $(TARGET)
