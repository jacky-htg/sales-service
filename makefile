init:
	go mod init sales
	
gen:
	protoc --proto_path=proto --go_out=paths=source_relative,:./pb --go-grpc_out=paths=source_relative,:./pb proto/*/*.proto
	
migrate:
	go run cmd/cli.go migrate
	
seed:
	go run cmd/cli.go seed
	
server:
	go run server.go
	
.PHONY: init gen migrate seed server