// SPDX-License-Identifier: AGPL-3.0-or-later

package db

import "io/fs"

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
