build:
	@go build -o ./bin/dfs

run : build
	@./bin/dfs

test :
	@go test ./...

#verbose test
testv :
	@go test ./... -v 