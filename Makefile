.PHONY: help dev build check fmt vet test race verify

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "  dev     启动开发模式（菜单栏应用，Alt+C 打开剪贴板面板）"
	@echo "  build   构建 macOS 应用，产物在 build/"
	@echo "  check   NSGo 检查：视图、桥接、打包配置改动后运行"
	@echo "  fmt     格式化检查"
	@echo "  vet     静态分析"
	@echo "  test    运行 Go 单元测试"
	@echo "  race    并发、剪贴板监听、数据库 worker 改动后运行"
	@echo "  verify  本地提交前完整检查（fmt + vet + test）"

dev:
	nsgo dev cmd/ecopaste

build:
	nsgo build cmd/ecopaste

check:
	nsgo check cmd/ecopaste

fmt:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

verify: fmt vet test
