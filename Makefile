.PHONY: build build-linux test clean

# 默认构建本地可执行文件（macOS）
build:
	go build -o ossre ./cmd/ossre

# 构建Linux可执行文件（用于部署）
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ossre-linux ./cmd/ossre

# 运行测试
test:
	go test ./...

# 清理构建产物
clean:
	rm -f ossre ossre-linux