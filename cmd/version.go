package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Version can be set via:
// -ldflags="-X 'github.com/rosskirkpat/kscale/cmd/version.Version=$TAG'"
var Version string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Long: `The version string is completely dependent on how the binary was built, so you should not depend on the version format. It may change without notice.
This could be an arbitrary string, if specified via -ldflags.
This could also be the go module version, if built with go modules (often "(dev)").`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if Version == "" {
			logrus.Errorf("could not determine build information")
		} else {
			fmt.Println(Version)
		}
	},
}
