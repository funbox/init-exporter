################################################################################

# This Makefile generated by GoMakeGen 1.1.2 using next command:
# gomakegen .
#
# More info: https://kaos.sh/gomakegen

################################################################################

.DEFAULT_GOAL := help
.PHONY = fmt all clean git-config deps deps-test test gen-fuzz help

################################################################################

all: init-exporter ## Build all binaries

init-exporter: ## Build init-exporter binary
	go build init-exporter.go

install: ## Install all binaries
	cp init-exporter /usr/bin/init-exporter

uninstall: ## Uninstall all binaries
	rm -f /usr/bin/init-exporter

git-config: ## Configure git redirects for stable import path services
	git config --global http.https://pkg.re.followRedirects true

deps: git-config ## Download dependencies
	go get -d -v pkg.re/essentialkaos/ek.v10
	go get -d -v pkg.re/essentialkaos/go-simpleyaml.v1

deps-test: git-config ## Download dependencies for tests
	go get -d -v pkg.re/check.v1
	go get -d -v pkg.re/essentialkaos/ek.v10

test: ## Run tests
	go test -covermode=count ./export ./procfile

gen-fuzz: ## Generate archives for fuzz testing
	which go-fuzz-build &>/dev/null || go get -u -v github.com/dvyukov/go-fuzz/go-fuzz-build
	go-fuzz-build -o procfile-fuzz.zip github.com/funbox/init-exporter/procfile

fmt: ## Format source code with gofmt
	find . -name "*.go" -exec gofmt -s -w {} \;

clean: ## Remove generated files
	rm -f init-exporter

help: ## Show this info
	@echo -e '\n\033[1mSupported targets:\033[0m\n'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[33m%-15s\033[0m %s\n", $$1, $$2}'
	@echo -e ''
	@echo -e '\033[90mGenerated by GoMakeGen 1.1.2\033[0m\n'

################################################################################
