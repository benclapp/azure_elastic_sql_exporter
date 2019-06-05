# Go parameters
VERSION=$(shell cat VERSION)
VERSION_MINOR=$(shell cat VERSION_MINOR)
GOCMD=go
GOBUILD=$(GOCMD) build
GOFLAGS=-ldflags "-X main.version=$(VERSION)"
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get

all: build-all docker
build: deps
		$(GOBUILD) $(GOFLAGS) -v azure_elastic_sql_exporter.go
clean:
		$(GOCLEAN)
		rm -rf bin
		rm -rf azure_elastic_sql_exporter-*
run:
		$(GORUN) $(GOFLAGS) ./...
deps:
		$(GOGET) -d -v ./...
build-all: deps
		mkdir -p bin
		for OS in linux darwin windows ; do \
			env GOOS=$$OS GOARCH=amd64 $(GOBUILD) $(GOFLAGS) azure_elastic_sql_exporter.go; \
			mkdir azure_elastic_sql_exporter-$(VERSION)-$$OS-amd64 ; \
			if [ -f azure_elastic_sql_exporter ] ; then mv azure_elastic_sql_exporter azure_elastic_sql_exporter-$(VERSION)-$$OS-amd64 ; fi ; \
			if [ -f azure_elastic_sql_exporter.exe ] ; then mv azure_elastic_sql_exporter.exe azure_elastic_sql_exporter-$(VERSION)-$$OS-amd64 ; fi ; \
			tar -czf azure_elastic_sql_exporter-$(VERSION)-$$OS-amd64.tar.gz azure_elastic_sql_exporter-$(VERSION)-$$OS-amd64 ; \
			mv azure_elastic_sql_exporter-$(VERSION)-$$OS-amd64.tar.gz bin/ ; \
			rm -rf azure_elastic_sql_exporter-$(VERSION)-$$OS-amd64 ; \
		done

docker:
		docker build --pull -t benclapp/azure_elastic_sql_exporter:$(VERSION) -t benclapp/azure_elastic_sql_exporter:latest .

docker-push:
		docker push benclapp/azure_elastic_sql_exporter:$(VERSION)
		docker push benclapp/azure_elastic_sql_exporter:latest
