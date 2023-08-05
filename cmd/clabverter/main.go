package main

import (
	"os"

	clabernetescmdclabvertercli "gitlab.com/carlmontanari/clabernetes/cmd/clabverter/cli"
)

func main() {
	err := clabernetescmdclabvertercli.Entrypoint().Run(os.Args)
	if err != nil {
		panic(err)
	}
}
