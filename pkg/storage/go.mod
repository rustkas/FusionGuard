module github.com/fusionguard/pkg/storage

go 1.21

require (
	github.com/fusionguard/pkg/telemetry v0.0.0
	github.com/lib/pq v1.10.9
)

replace github.com/fusionguard/pkg/telemetry => ../telemetry
