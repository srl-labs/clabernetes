package main

import (
	"os"

	clabernetescmdclabvertercli "github.com/srl-labs/clabernetes/cmd/clabverter/cli"
)

func main() {
	err := clabernetescmdclabvertercli.Entrypoint().Run(os.Args)
	if err != nil {
		panic(err)
	}
}
