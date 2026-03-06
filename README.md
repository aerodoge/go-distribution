# go-distribution

项目结构参考：https://github.com/golang-standards/project-layout

```
myproject/
├── cmd/                    # 各个可执行程序的入口
│   ├── server/
│   │   └── main.go
│   └── worker/
│       └── main.go
│
├── internal/               # 私有代码，外部包无法导入（编译器强制）
│   ├── service/            # 业务逻辑
│   ├── repository/         # 数据库访问
│   ├── model/              # 数据模型
│   ├── api/                # HTTP handler + router
│   ├── middleware/         # 中间件
│   └── config/             # 配置结构体
│
├── pkg/                    # 可被外部项目导入的公共库
│   ├── database/
│   ├── redis/
│   └── jwt/
│
├── config/                 # 配置文件（yaml/toml）
├── migrations/             # 数据库迁移 SQL
├── docs/                   # 文档
├── scripts/                # 构建/部署脚本
├── go.mod
└── go.sum
```

## Test

```
运行全部测试：
go test ./pkg/utils/

运行单个测试（用 -run 指定测试函数名）：
go test ./pkg/utils/ -run TestNextID_ClockRollback

加 -v 可以看详细输出，加 -race 可以开启竞态检测（并发测试推荐加）：
go test ./pkg/utils/ -v -race
```