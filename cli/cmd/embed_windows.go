package cmd

import (
	"embed"
	_ "embed"
)

//go:embed backend.exe
var backendBinary []byte

//go:embed frontend
var frontendFiles embed.FS
