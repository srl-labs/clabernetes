package main

import (
	"os"

	clabernetescmdclabernetescli "github.com/srl-labs/clabernetes/cmd/clabernetes/cli"
)

func main() {
	err := clabernetescmdclabernetescli.Entrypoint().Run(os.Args)
	if err != nil {
		panic(err)
	}
}
