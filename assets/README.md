# Assets

## `banner.txt`

The Atomwright wordmark, rendered verbatim wherever a banner is shown.

Edit the file to change the banner. Nothing needs recompiling — the previous
implementation hardcoded its ASCII art in a Go source file, which meant a
rebuild for every tweak.

Rules for whatever replaces it:

- **41 columns or fewer.** 80 is the safe terminal floor, and a banner that
  overflows wraps into unreadable fragments. The current wordmark is 41.
- No trailing whitespace, and a single trailing newline.
- Plain text only. The file is printed as-is, so anything in it is displayed,
  including comments — there is no comment syntax.
- Colour belongs to the renderer, not to this file. Keep it uncoloured so the
  caller can apply a theme.
