package frontend

import "embed"

//go:embed index.html *.js *.css lib
var Assets embed.FS
