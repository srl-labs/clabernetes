package cli

import (
	clabernetesclicker "github.com/srl-labs/clabernetes/clicker"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslauncher "github.com/srl-labs/clabernetes/launcher"
	clabernetesmanager "github.com/srl-labs/clabernetes/manager"
	"github.com/urfave/cli/v2"
)

const (
	// indicates the manager command is being run in init/initialization mode (as in, in an init
	// container).
	cliInitializer = "initializer"

	// indicates the clicker invocation should target all nodes; if unset we only target nodes that
	// do *not* have the LabelClickerNodeConfigured label set. Note that this is applied after the
	// selector (if present), so it's not technically "all" nodes, its all nodes that were selected.
	clickerOverrideNodes = "overrideNodes"

	// indicates the node selector filter that should be applied to the nodes clicker targets.
	clickerNodeSelector = "nodeSelector"
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
			{
				Name:  "clicker",
				Usage: "run the node clicker",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: clickerOverrideNodes,
						Usage: "indicates if the clicker should be re-ran on all nodes" +
							" even if they already have a clabernetes clicker label",
						Required: false,
						Value:    false,
					},
					&cli.StringFlag{
						Name: clickerNodeSelector,
						Usage: "node selector to target which nodes clicker should" +
							" execute on",
						Required: false,
						Value:    "",
					},
				},
				Action: func(c *cli.Context) error {
					clabernetesclicker.StartClabernetes(
						c.Bool(clickerOverrideNodes),
						c.String(clickerNodeSelector),
					)

					return nil
				},
			},
		},
	}
}
