lint: protolint golint

protobuf:
	protoc -I . --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/parser/parser.proto

	protoc -I . --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/repository/repository.proto

	protoc -I . --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/topology/topology.proto

protolint:
	protolint .

golint:
	golangci-lint run -E gocritic -v ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

test:
	go test ./...