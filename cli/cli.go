package cli

import (
	"github.com/urfave/cli/v2"
	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	claberneteslauncher "gitlab.com/carlmontanari/clabernetes/launcher"
	clabernetesmanager "gitlab.com/carlmontanari/clabernetes/manager"
)

const (
	cliInitializer = "initializer"
)

// Entrypoint returns the clabernetes manager entrypoint, kicking off one of the clabernetes
// processes.
func Entrypoint() *cli.App {
	cli.VersionPrinter = ShowVersion

	return &cli.App{
		Name:    "clabernetes",
		Version: clabernetesconstants.Version,
		Usage:   "run clabernetes manager",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "run the manager",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:     cliInitializer,
						Usage:    "indicate if this instance should run initialization",
						Required: false,
						Value:    false,
					},
				},
				Action: func(c *cli.Context) error {
					clabernetesmanager.StartClabernetes(
						c.Bool(cliInitializer),
					)

					return nil
				},
			},
			{
				Name:  "launch",
				Usage: "run the launcher",
				Flags: []cli.Flag{},
				Action: func(c *cli.Context) error {
					claberneteslauncher.StartClabernetes()

					return nil
				},
			},
		},
	}
}
