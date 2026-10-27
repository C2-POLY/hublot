// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build embedweb

package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the SPA built into the binary (go build -tags embedweb, after npm run build).
func Dist() (fs.FS, error) { return fs.Sub(dist, "dist") }
