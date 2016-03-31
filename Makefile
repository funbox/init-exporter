########################################################################################

DESTDIR?=
PREFIX?=/usr

########################################################################################

.PHONY = all clean install uninstall deps test

########################################################################################

all: bin

deps:
	go get -v pkg.re/check.v1
	go get -v pkg.re/essentialkaos/ek.v1
	go get -v github.com/smallfish/simpleyaml
	go get -v gopkg.in/yaml.v2

bin:
	go build init-exporter.go

test:
	go test ./...

install:
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp init-exporter $(DESTDIR)$(PREFIX)/bin/

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/init-exporter

clean:
	rm -f init-exporter
