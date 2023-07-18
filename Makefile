.PHONY: verify vendor build proto build-novendor clean docker help client

protoc: ## build protobuf
	docker run --rm --user `id -u ${USER}` -v `pwd`:`pwd` -w `pwd` znly/protoc -I=./protobuf  --go_out=plugins=grpc:.  ./protobuf/*

server: ## build binarys
	go build -mod=vendor -o bin/tss-server ./cmd/main.go

client: ## build client
	go build -mod=vendor -o bin/keygen ./client/keygen.go
	go build -mod=vendor -o bin/signing ./client/signing.go
	go build -mod=vendor -o bin/resharing ./client/resharing.go
	go build -mod=vendor -o bin/vss ./client/vss.go

clean: ## delete  the build target
	rm bin/*

fmt: ## format golang code
	go fmt ./cmd/...
	go fmt ./client/...
	go fmt ./pkg/...

docker: ## docker build image
	docker build -t tss-lib:grpc .
