BIN      := check_redis_go 
OSARCH   := "darwin/amd64 linux/amd64 windows/amd64"


all: build

test: deps build
	./test.sh

deps:
	go get -d -v -t ./...
	go get github.com/golang/lint/golint
	go get github.com/mitchellh/gox

lint: deps
	go vet ./...
	golint -set_exit_status ./...

package:
	rm -fR ./pkg && mkdir ./pkg ;\
		gox \
		-osarch $(OSARCH) \
		-output "./pkg/{{.OS}}_{{.Arch}}/{{.Dir}}" \
		./cmd/...;\
	    for d in $$(ls ./pkg);do zip ./pkg/$${d}.zip ./pkg/$${d}/*;done

build:
	go build 

linuxbuild:
	GOOS=linux GOARCH=amd64 make build

clean:
	go clean
