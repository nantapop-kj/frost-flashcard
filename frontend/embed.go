package frontend

import "embed"

//go:embed index.html *.js lib
var Assets embed.FS
