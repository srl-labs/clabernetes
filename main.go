package main

import (
	"os"

	clabernetescli "gitlab.com/carlmontanari/clabernetes/cli"
)

func main() {
	err := clabernetescli.Entrypoint().Run(os.Args)
	if err != nil {
		panic(err)
	}
}
