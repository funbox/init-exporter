########################################################################################

DESTDIR?=
PREFIX?=/usr

########################################################################################

.PHONY = all clean install uninstall deps test

########################################################################################

all: deps bin

deps:
	go get -v pkg.re/check.v1
	go get -v pkg.re/essentialkaos/ek.v5
	go get -v github.com/smallfish/simpleyaml
	go get -v gopkg.in/yaml.v2

bin:
	go build init-exporter.go

fmt:
	find . -name "*.go" -exec gofmt -s -w {} \;

test:
	go test ./...

install:
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp init-exporter $(DESTDIR)$(PREFIX)/sbin/
	cp common/init-exporter.conf $(DESTDIR)/etc/

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/sbin/init-exporter
	rm -rf $(DESTDIR)/etc/init-exporter.conf

clean:
	rm -f init-exporter

upstart-playground:
	docker build -f ./Dockerfile.upstart -t upstart-playground . && docker run -ti --rm=true upstart-playground /bin/bash

systemd-playground:
	docker build -f ./Dockerfile.systemd -t systemd-playground . && docker run -ti --rm=true systemd-playground /bin/bash
