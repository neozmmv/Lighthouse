/* Lighthouse - Temporary file-receiving station.
   Copyright (C) 2026 neozmmv

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published
   by the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>. */

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "lighthouse",
	Short: "A temporary file-receiving station on the Tor Network.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// skip checks for help and version commands
		skipCommands := []string{"help", "version", "update"}
		for _, name := range skipCommands {
			if cmd.Name() == name {
				return nil
			}
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
