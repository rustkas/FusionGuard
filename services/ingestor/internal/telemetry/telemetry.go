package telemetry

import "fmt"

type SampleQuality string

const (
    QualityGood SampleQuality = "good"
    QualityMissing SampleQuality = "missing"
    QualityOutlier SampleQuality = "outlier"
)

type ChannelSample struct {
    Name    string        `json:"name"`
    Value   float64       `json:"value"`
    Quality SampleQuality `json:"quality"`
}

type TelemetryPoint struct {
    ShotID   string           `json:"shot_id"`
    TsUnixNs int64            `json:"ts_unix_ns"`
    Channels []ChannelSample  `json:"channels"`
}

type IngestAck struct {
    ShotID         string `json:"shot_id"`
    AcceptedPoints int    `json:"accepted_points"`
}

func (t *TelemetryPoint) Valid() error {
    if t.ShotID == "" {
        return fmt.Errorf("missing shot_id")
    }
    if len(t.Channels) == 0 {
        return fmt.Errorf("no channels")
    }
    if t.TsUnixNs <= 0 {
        return fmt.Errorf("invalid timestamp")
    }
    return nil
}
