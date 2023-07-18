# Configuration
```shell
router:     # TSS server ports 
  - ':8000'
  - ':8001'
  - ':8002'
savepath: "keygen_data.json" # share save path（all parties use the same path. Not used yet.）

partyNum: 3  #party number
threshold: 1 # threshold
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