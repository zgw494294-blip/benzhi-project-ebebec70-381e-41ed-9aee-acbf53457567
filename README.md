# 口述史授权清理服务

本项目为口述史馆员和伦理复核员提供版本化 JSON HTTP API，用于登记访谈转录稿、扫描敏感信息、记录人工裁定、生成候选稿、独立复核并发布不可变的授权公开版本。

## 构建、运行和测试

```text
go test ./...
go run ./cmd/oralclear -addr=127.0.0.1:19081
go run ./cmd/oralclear -selfcheck -addr=127.0.0.1:19081
```

监听地址可通过 `-addr` 或 `PORT` 环境变量配置，默认仅绑定 `127.0.0.1:19081`。数据保存在工作目录的 SQLite 文件 `oralclear.db`。
