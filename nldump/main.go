// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/platinasystems/go/goes/cmd"
	"github.com/platinasystems/go/goes/cmd/nldump"
)

var Exit = os.Exit
var Stderr io.Writer = os.Stderr

func main() {
	err := cmd.New(nldump.New()).Main()
	if err != nil {
		fmt.Fprintln(Stderr, err)
		Exit(1)
	}
}
