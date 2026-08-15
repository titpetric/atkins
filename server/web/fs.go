package web

import "embed"

// files holds the page templates. They are embedded so the server is
// still a single binary you can drop into a scratch image.
//
//go:embed templates/*.html
var files embed.FS
