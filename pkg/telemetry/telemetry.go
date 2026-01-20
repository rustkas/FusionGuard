package telemetry

type SampleQuality string

const (
	QualityGood    SampleQuality = "good"
	QualityMissing SampleQuality = "missing"
	QualityOutlier SampleQuality = "outlier"
)

type ChannelSample struct {
	Name    string        `json:"name"`
	Value   float64       `json:"value"`
	Quality SampleQuality `json:"quality"`
}

type TelemetryPoint struct {
	ShotID   string          `json:"shot_id"`
	TsUnixNs int64           `json:"ts_unix_ns"`
	Channels []ChannelSample `json:"channels"`
}

func (t *TelemetryPoint) Valid() error {
	if t.ShotID == "" {
		return ErrMissingShotID
	}
	if len(t.Channels) == 0 {
		return ErrNoChannels
	}
	if t.TsUnixNs <= 0 {
		return ErrInvalidTimestamp
	}
	return nil
}

var (
	ErrMissingShotID    = &validationError{"missing shot_id"}
	ErrNoChannels       = &validationError{"no channels"}
	ErrInvalidTimestamp = &validationError{"invalid timestamp"}
)

type validationError struct {
	msg string
}

func (v *validationError) Error() string {
	return v.msg
}

type IngestAck struct {
	ShotID         string `json:"shot_id"`
	AcceptedPoints int    `json:"accepted_points"`
}
