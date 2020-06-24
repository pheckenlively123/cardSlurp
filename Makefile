GOCMD=go
GOBLD=$(GOCMD) build
TARGET=$(GOPATH)/bin

cardSlurp: cardSlurp.go cardfileutil/*.go filecontrol/*.go
	$(GOBLD) cardSlurp.go

all: cardSlurp

clean:
	rm cardSlurp

install: all
	mkdir -p $(TARGET)
	cp cardSlurp $(TARGET)
