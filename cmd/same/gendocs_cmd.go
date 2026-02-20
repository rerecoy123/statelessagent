package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func gendocsCmd() *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:    "gendocs",
		Short:  "Generate documentation (man pages, markdown)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}
			header := &doc.GenManHeader{
				Title:   "SAME",
				Section: "1",
				Source:  "SAME " + Version,
			}
			return doc.GenManTree(cmd.Root(), header, outDir)
		},
	}
	cmd.Flags().StringVar(&outDir, "dir", "docs/man", "output directory for man pages")
	return cmd
}
