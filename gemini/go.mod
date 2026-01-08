module github.com/zoobzio/zyn/gemini

go 1.24

toolchain go1.25.3

replace github.com/zoobzio/zyn => ../

require (
	github.com/zoobzio/capitan v0.1.0
	github.com/zoobzio/zyn v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/zoobzio/clockz v0.0.2 // indirect
	github.com/zoobzio/pipz v0.1.3 // indirect
	github.com/zoobzio/sentinel v0.1.1 // indirect
)
