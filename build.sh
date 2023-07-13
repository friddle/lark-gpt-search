cd $(dirname $0)

go mod tidy
go mod download
go build -o dist/feishu_gpt_search ./main.go
