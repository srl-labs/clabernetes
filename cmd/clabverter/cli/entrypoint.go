package cli

import (
	"github.com/urfave/cli/v2"
	clabernetesclabverter "gitlab.com/carlmontanari/clabernetes/clabverter"
	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	claberneteslogging "gitlab.com/carlmontanari/clabernetes/logging"
)

const (
	topologyFile         = "topologyFile"
	outputDirectory      = "outputDirectory"
	destinationNamespace = "destinationNamespace"
	insecureRegistries   = "insecureRegistries"
	debug                = "debug"
	quiet                = "quiet"
	stdout               = "stdout"
)

// Entrypoint returns the clabernetes clabverter entrypoint.
func Entrypoint() *cli.App {
	cli.VersionPrinter = ShowVersion

	return &cli.App{
		Name:    clabernetesconstants.Clabverter,
		Version: clabernetesconstants.Version,
		Usage:   "run clabernetes clabverter -- clab to clabernetes manifest(s) converter",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     topologyFile,
				Usage:    "set the topology file to parse",
				Required: false,
				Value:    "topo.yaml",
			},
			&cli.StringFlag{
				Name:     outputDirectory,
				Usage:    "set the output directory for the converted manifest(s)",
				Required: false,
				Value:    "converted",
			},
			&cli.StringFlag{
				Name:     destinationNamespace,
				Usage:    "set the namespace for the rendered manifest(s)",
				Required: false,
				Value:    "clabernetes",
			},
			&cli.StringFlag{
				Name:     insecureRegistries,
				Usage:    "comma separated list of insecure registries",
				Required: false,
				Value:    "",
			},
			&cli.BoolFlag{
				Name:     debug,
				Usage:    "enable debug logging",
				Required: false,
				Value:    false,
			},
			&cli.BoolFlag{
				Name:     quiet,
				Usage:    "disable all output (other than stdout if enabled)",
				Required: false,
				Value:    false,
			},
			&cli.BoolFlag{
				Name: stdout,
				Usage: "print all rendered manifests to stdout, if set," +
					" output not written to disk",
				Required: false,
				Value:    false,
			},
		},
		Action: func(c *cli.Context) error {
			err := clabernetesclabverter.MustNewClabverter(
				c.String(topologyFile),
				c.String(outputDirectory),
				c.String(destinationNamespace),
				c.String(insecureRegistries),
				c.Bool(debug),
				c.Bool(quiet),
				c.Bool(stdout),
			).Clabvert()

			claberneteslogging.GetManager().Flush()

			return err
		},
	}
}
