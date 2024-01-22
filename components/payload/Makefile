pb:
	protoc -I=. --go_out=:./ --go_opt=paths=source_relative ./*.proto

install-plugins:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
