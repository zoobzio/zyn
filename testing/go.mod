module github.com/zoobzio/zyn/testing

go 1.24

toolchain go1.25.3

replace github.com/zoobzio/zyn => ../

replace github.com/zoobzio/zyn/openai => ../openai

require (
	github.com/zoobzio/zyn v0.0.0-00010101000000-000000000000
	github.com/zoobzio/zyn/openai v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/zoobzio/capitan v1.0.0 // indirect
	github.com/zoobzio/clockz v1.0.0 // indirect
	github.com/zoobzio/pipz v1.0.4 // indirect
	github.com/zoobzio/sentinel v1.0.2 // indirect
)
