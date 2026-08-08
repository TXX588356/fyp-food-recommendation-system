package cmd

import (
	"context"
	"fyp/food-rs/app/cmd/server"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "backend",
	Short: "backend CLI",
	Long:  "backend CLI Handler",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.Run(cmd.Context())
	},
}

func Execute(ctx context.Context) error {
	rootCmd.SetContext(ctx)
	return rootCmd.Execute()
}
