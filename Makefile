CURRENT=$(shell pwd)
OSNAME=linux
ARCHNAME=amd64
CONTAINER_NAME=fluent-bit-pubsub-custom
GOLANG_IMAGE=golang:1.22.5

local-build:
	go build -buildmode=c-shared -o pubsub.so .

local-build-fast:
	go build pubsub.go

build-linux:
	@if [ $$(docker images -q $(GOLANG_IMAGE) | wc -l) -eq 0 ]; then \
		docker pull $(GOLANG_IMAGE); \
	fi
	@if [ $$(docker ps -a | grep $(CONTAINER_NAME) | wc -l) -eq 1 ]; then \
		docker rm -f $(CONTAINER_NAME); \
	fi
	docker run -itd --platform ${OSNAME}/${ARCHNAME} -v $(CURRENT):/build --name $(CONTAINER_NAME) $(GOLANG_IMAGE) /bin/bash
	docker exec $(CONTAINER_NAME) sh -c "cd /build && go build -buildmode=c-shared -o pubsub.so ."
	docker kill $(CONTAINER_NAME)

build-linux-fast:
	@if [ $$(docker images -q $(GOLANG_IMAGE) | wc -l) -eq 0 ]; then \
		docker pull $(GOLANG_IMAGE); \
	fi
	@if [ $$(docker ps -a | grep $(CONTAINER_NAME) | wc -l) -eq 0 ]; then \
		docker run -itd --platform $(OSNAME)/$(ARCHNAME) -v $(CURRENT):/build --name $(CONTAINER_NAME) $(GOLANG_IMAGE) /bin/bash; \
	elif [ $$(docker ps | grep $(CONTAINER_NAME) | wc -l) -eq 0 ]; then \
		docker start $(CONTAINER_NAME); \
	fi
	docker exec $(CONTAINER_NAME) sh -c "cd /build && go build -buildmode=c-shared -o pubsub.so ."
	docker stop $(CONTAINER_NAME)

test:
	go test -v -race  -coverprofile=coverage.txt -covermode=atomic ./...

cover:
	go tool cover -html=coverage.txt
	
clean:
	@if [ $$(docker ps -a | grep $(CONTAINER_NAME) | wc -l) -eq 1 ]; then \
		docker rm -f $(CONTAINER_NAME); \
	fi
	rm -rf *.so *.h *~
