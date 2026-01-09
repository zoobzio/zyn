module github.com/zoobzio/zyn/tendo

go 1.25.0

toolchain go1.25.3

replace github.com/zoobzio/zyn => ../

replace github.com/zoobzio/tendo => ../../tendo

replace github.com/zoobzio/tendo/cpu => ../../tendo/cpu

replace github.com/zoobzio/tendo/cuda => ../../tendo/cuda

require (
	github.com/zoobzio/capitan v1.0.0
	github.com/zoobzio/tendo v0.1.0
	github.com/zoobzio/tendo/cpu v0.0.0-20260106225412-f59220386cbd
	github.com/zoobzio/tendo/cuda v0.0.0-20260106225412-f59220386cbd
	github.com/zoobzio/zyn v0.0.0-00010101000000-000000000000
)

require (
	github.com/daulet/tokenizers v1.24.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/zoobzio/clockz v1.0.0 // indirect
	github.com/zoobzio/pipz v1.0.4 // indirect
	github.com/zoobzio/sentinel v1.0.2 // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
)
