# BENZHI_README

基于 Go 实现的口述史授权清理服务 HTTP API 项目，一款后端服务，口述史馆员与伦理复核员完成访谈转录稿清理和授权发布。

## 项目说明
- 项目：benzhi-project-ebebec70-381e-41ed-9aee-acbf53457567
- 项目用途：口述史馆员与伦理复核员完成访谈转录稿清理和授权发布。
- Go 工具链：`golang:1.22`
- 前端工具链：无

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/oralclear -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-ebebec70-381e-41ed-9aee-acbf53457567-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-ebebec70-381e-41ed-9aee-acbf53457567-arm64 linux/arm64
docker run -it benzhi-project-ebebec70-381e-41ed-9aee-acbf53457567-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/oralclear -selfcheck -addr=127.0.0.1:19081`
