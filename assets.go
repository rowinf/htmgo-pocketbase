//go:build !prod
// +build !prod

package main

import (
	"io/fs"
	"htmgopocketbase/internal/embedded"
)

func GetStaticAssets() fs.FS {
	return embedded.NewOsFs()
}
