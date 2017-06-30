package cmd

import (
	"github.com/getgauge/gauge/logger"
	"github.com/getgauge/gauge/search"
	"github.com/getgauge/gauge/validation"
	"github.com/spf13/cobra"
)

var (
	searchCmd = &cobra.Command{
		Use:   "search [flags] [query] [args]",
		Short: "Search for text in specs.",
		Long:  "Search for text in specs.",
		Example: `  gauge search \"search string\" specs/
  gauge search index specs/`,
		Run: func(cmd *cobra.Command, args []string) {
			setGlobalFlags()
			if index && len(args) < 1 {
				logger.Fatalf("Error: Missing argument <specs dir>.\n%s", cmd.UsageString())
			} else if len(args) < 1 {
				logger.Fatalf("Error: Missing argument(s) <query>.\n")
			}
			if err := isValidGaugeProject(args[1:]); err != nil {
				logger.Fatalf(err.Error())
			}
			initPackageFlags()
			if index {
				res := validation.ValidateSpecs(args, false)
				search.Initialize(res.SpecCollection)
			} else {
				search.Search(args[0])
			}
		},
	}
	index bool
)

func init() {
	GaugeCmd.AddCommand(searchCmd)
	searchCmd.Flags().BoolVarP(&index, "index", "i", false, "Index gauge specs.")
}
