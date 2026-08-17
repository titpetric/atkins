# Vendored browser assets

The terminal page needs a terminal emulator, and a build log is full of
the escape sequences that need one: atkins draws its tree by moving the
cursor and clearing lines, so anything less than an emulator renders a
three-line tree as several hundred lines of noise.

These files are that emulator, checked in rather than fetched.

| File                  | Package             | Version | Licence |
|-----------------------|---------------------|---------|---------|
| `xterm.js`            | `@xterm/xterm`      | 5.5.0   | MIT     |
| `xterm.css`           | `@xterm/xterm`      | 5.5.0   | MIT     |
| `xterm-addon-fit.js`  | `@xterm/addon-fit`  | 0.10.0  | MIT     |

`LICENSE.xterm` is the licence all three are published under.

They are committed for the same reason the templates are compiled: a
server should build and run from a checkout and a Go toolchain. A CDN
would put a third party between an operator and their own CI, break an
instance on a private network, and make the page a request to somebody
else every time it is opened. There is no build step here either — the
files are the published distributions, embedded with `go:embed` and
served from `/assets/`.

The `.map` files the distributions reference are deliberately not here.
They are only ever requested with developer tools open, they are larger
than the code they annotate, and nothing in this repository is going to
be debugged through them — so the request 404s, which is the honest
answer.

Updating means replacing a file and its row in the table above:

```bash
curl -sSf -o xterm.js  https://cdn.jsdelivr.net/npm/@xterm/xterm@<version>/lib/xterm.js
curl -sSf -o xterm.css https://cdn.jsdelivr.net/npm/@xterm/xterm@<version>/css/xterm.css
curl -sSf -o xterm-addon-fit.js https://cdn.jsdelivr.net/npm/@xterm/addon-fit@<version>/lib/addon-fit.js
```
