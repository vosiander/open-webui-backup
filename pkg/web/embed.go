package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var dist embed.FS

// GetDistFS returns the embedded filesystem for the frontend dist directory
func GetDistFS() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
