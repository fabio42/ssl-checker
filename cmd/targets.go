package cmd

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var fileTargetsCmd = &cobra.Command{
	Use:   "files [flags] fileTargets1[,fileTargets2,fileTargets3...]",
	Short: "Run test against provided list of files",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetsList := strings.Split(args[0], ",")
		targets := make(map[string]string, len(targetsList))
		for _, e := range targetsList {
			targets[filepath.Base(e)] = e
		}
		runQueries(targets, map[string][]string{})
	},
}

func init() {
	rootCmd.AddCommand(fileTargetsCmd)
}

var domainTargetsCmd = &cobra.Command{
	Use:   "domains [flags] domain1[,domain2,domain3...]",
	Short: "Run test against provided list of domains",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetsList := strings.Split(args[0], ",")

		targets := make(map[string][]string, 1)
		targets["customDomains"] = targetsList

		runQueries(map[string]string{}, targets)
	},
}

func init() {
	rootCmd.AddCommand(domainTargetsCmd)
}
