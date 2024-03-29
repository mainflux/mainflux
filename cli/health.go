// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import "github.com/spf13/cobra"

// NewHealthCmd returns health check command.
func NewHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health <service>",
		Short: "Health Check",
		Long: "Mainflux service Health Check\n" +
			"usage:\n" +
			"\tmainflux-cli health <service>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}
			v, err := sdk.Health(args[0])
			if err != nil {
				logError(err)
				return
			}

			logJSON(v)
		},
	}
}
