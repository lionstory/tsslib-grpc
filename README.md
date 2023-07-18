# Configuration
```shell
router:     #终端路由配置
  - ':8000'
  - ':8001'
  - ':8002'
savepath: "keygen_data.json" #保存分片文件地址（目前还未使用该配置，三方共同一个目录）

partyNum: 3  #party数量
threshold: 1 # 门限阈值
```

# Start service
```shell
go mod tidy
```
```shell
go run cmd/main.go -p 8000 -c conf/config.yaml
go run cmd/main.go -p 8001 -c conf/config.yaml
go run cmd/main.go -p 8002 -c conf/config.yaml
```
# Test
## Trigger keygen
```shell
go run client/keygen.go
```

## Trigger signing
```shell
go run client/signing.go
```