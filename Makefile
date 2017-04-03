########################################################################################

DESTDIR?=
PREFIX?=/usr

########################################################################################

.PHONY = all clean install uninstall deps deps-glide test upstart-playground systemd-playground

########################################################################################

all: init-exporter init-exporter-converter

init-exporter:
	go build init-exporter.go

init-exporter-converter:
	go build init-exporter-converter.go

deps:
	go get -d -v pkg.re/check.v1
	go get -d -v pkg.re/essentialkaos/ek.v7
	go get -d -v pkg.re/essentialkaos/go-simpleyaml.v1
	go get -d -v pkg.re/yaml.v2

deps-glide:
	glide install

fmt:
	find . -name "*.go" -exec gofmt -s -w {} \;

test:
	go test ./procfile ./export -covermode=count

install:
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp init-exporter $(DESTDIR)$(PREFIX)/bin/
	cp init-exporter-converter $(DESTDIR)$(PREFIX)/bin/
	cp common/init-exporter.conf $(DESTDIR)/etc/

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/init-exporter
	rm -f $(DESTDIR)$(PREFIX)/bin/init-exporter-converter
	rm -rf $(DESTDIR)/etc/init-exporter.conf

clean:
	rm -f init-exporter
	rm -f init-exporter-converter

upstart-playground:
	docker build -f ./Dockerfile.upstart -t upstart-playground . && docker run -ti --rm=true upstart-playground /bin/bash

systemd-playground:
	docker build -f ./Dockerfile.systemd -t systemd-playground . && docker run -ti --rm=true systemd-playground /bin/bash
