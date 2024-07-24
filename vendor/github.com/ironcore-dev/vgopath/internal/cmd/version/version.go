// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"io"

	"github.com/ironcore-dev/vgopath/internal/version"
	"github.com/spf13/cobra"
)

func Command(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints version information.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(out)
		},
	}

	return cmd
}

func Run(out io.Writer) error {
	version.FPrint(out)
	return nil
}
