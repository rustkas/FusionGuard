module github.com/fusionguard/services/api_gateway

go 1.21

require (
    github.com/fusionguard/pkg/storage v0.0.0
    github.com/prometheus/client_golang v1.16.0
    gopkg.in/yaml.v3 v3.0.1
)

replace github.com/fusionguard/pkg/storage => ../../pkg/storage
