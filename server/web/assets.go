package web

import (
	"embed"
	"errors"
	"net/http"
	"strings"
)

// assets are the files the terminal page loads: a terminal emulator and
// its stylesheet. See assets/README.md for what they are and why they
// are committed rather than fetched.
//
//go:embed assets/xterm.js assets/xterm.css assets/xterm-addon-fit.js
var assets embed.FS

// assetMaxAge is how long a browser may keep one of these.
//
// A year, because the content never changes: replacing a version means
// replacing a file, and a stale cache of the old one is only ever served
// to a page that was asking for the old one. It is the stylesheet inline
// in every other page that has to stay fresh, and that one is not here.
const assetMaxAge = "public, max-age=31536000, immutable"

// Asset serves a vendored browser asset.
//
// It is a hand-written handler rather than http.FileServer over the
// embedded FS because the FS holds a README and a licence too, and a
// server should serve what it meant to serve. The allowlist is three
// entries long and says so.
func (h *Handlers) Asset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/assets/")

	contentType, served := assetTypes[name]
	if !served {
		h.fail(w, r, http.StatusNotFound, errNoSuchAsset)
		return
	}

	contents, err := assets.ReadFile("assets/" + name)
	if err != nil {
		h.fail(w, r, http.StatusNotFound, errNoSuchAsset)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", assetMaxAge)
	// These are scripts and a stylesheet, and they are exactly what they
	// say they are. Nothing here should ever be sniffed into something
	// else.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	_, _ = w.Write(contents)
}

// assetTypes is both the allowlist and the media types, so a file cannot
// be served without somebody having decided what it is.
var assetTypes = map[string]string{
	"xterm.js":           "text/javascript; charset=utf-8",
	"xterm-addon-fit.js": "text/javascript; charset=utf-8",
	"xterm.css":          "text/css; charset=utf-8",
}

// errNoSuchAsset covers both "not in the allowlist" and "not embedded".
var errNoSuchAsset = errors.New("no such asset")
