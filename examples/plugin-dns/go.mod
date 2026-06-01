module github.com/datum-cloud/datumctl-examples/plugin-dns

go 1.25.8

require (
	github.com/spf13/cobra v1.10.2
	go.datum.net/datumctl v0.0.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
)

replace go.datum.net/datumctl => ../..
