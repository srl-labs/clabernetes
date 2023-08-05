package cli

import (
	"fmt"

	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"

	"github.com/urfave/cli/v2"
)

// ShowVersion shows the clabernetes version information for clabernetes CLI tools.
func ShowVersion(_ *cli.Context) {
	fmt.Printf("\tversion: %s\n", clabernetesconstants.Version)                  //nolint:forbidigo
	fmt.Printf("\tsource: %s\n", "https://gitlab.com/carlmontanari/clabernetes") //nolint:forbidigo
}
