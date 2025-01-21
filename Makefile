build:
	@go build -o ./bin/fs

run : build
	@./bin/dfs

test :
	@go test ./... -v