// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
)

func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil || info.Main.Version == "" {
		return "(unknown)"
	}
	return info.Main.Version
}

func FPrint(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Version: %s\n", Version())
}

func Print() {
	FPrint(os.Stdout)
}
