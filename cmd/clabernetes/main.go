package main

import (
	"os"

	clabernetescmdclabernetescli "gitlab.com/carlmontanari/clabernetes/cmd/clabernetes/cli"
)

func main() {
	err := clabernetescmdclabernetescli.Entrypoint().Run(os.Args)
	if err != nil {
		panic(err)
	}
}
