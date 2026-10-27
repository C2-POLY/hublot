// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !embedweb

// Package web carries the built SPA into the Go binary.
package web

import "io/fs"

// Dist returns nil without the embedweb build tag: development builds serve the
// SPA from web_dir or through the Vite dev server instead.
func Dist() (fs.FS, error) { return nil, nil }
