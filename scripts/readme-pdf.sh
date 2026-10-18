#!/bin/sh
# Builds a PDF of a Markdown document with pandoc and Prince, using
# docs/print.css.
#
#   scripts/readme-pdf.sh README.md            # README.pdf beside it
#   scripts/readme-pdf.sh docs/REFERENCE.md    # docs/REFERENCE.pdf
#   scripts/readme-pdf.sh README.md out.pdf
set -eu

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
	echo "usage: $0 FILE.md [out.pdf]" >&2
	exit 2
fi

root=$(cd "$(dirname "$0")/.." && pwd)
readme=$1
out=${2:-${readme%.md}.pdf}

title=$(sed -n 's/^# //p' "$readme" | head -n 1)
tmp=$(mktemp -d)
trap 'rm -rf "${tmp:?}"' EXIT
html=$tmp/readme.html

mkdir -p "$(dirname "$out")"
# print.css is linked from the page, after pandoc's own styles, so that it
# takes precedence over them (a style sheet given to prince --style wouldn't).
pandoc "$readme" --from gfm --standalone --metadata pagetitle="${title:-README}" \
	--css "file://$root/docs/print.css" --output "$html"
prince "$html" --no-warn-css --output "$out"
echo "$out"
