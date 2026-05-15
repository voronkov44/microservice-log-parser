.PHONY: lint protobuf protolint golint fmt tidy test test-unit test-integration

lint: protolint golint

protobuf:
	protoc -I proto --go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		parser/parser.proto

	protoc -I proto --go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		repository/repository.proto

	protoc -I proto --go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		topology/topology.proto

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

test-unit:
	go test ./...

test-integration:
	go test -tags=integration ./repository/adapters/db
