package cmd

import (
	"embed"
	_ "embed"
)

//go:embed lighthouse_server.exe
var backendBinary []byte

//go:embed frontend
var frontendFiles embed.FS
