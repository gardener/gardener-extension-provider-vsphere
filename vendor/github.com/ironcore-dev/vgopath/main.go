// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"

	"github.com/ironcore-dev/vgopath/internal/cmd/vgopath"
)

func main() {
	if err := vgopath.Command().Execute(); err != nil {
		log.Fatalln(err.Error())
	}
}
