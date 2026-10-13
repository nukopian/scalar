package scalar

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Pack and Unpack are a subset of Perl's pack and unpack: hex strings (H
// and h), bit strings (B and b), and floating-point numbers (f, d and F).
// That covers turning bytes into hex or bits and back, and numbers into
// their binary forms and back. Other template letters panic.
//
// The packed result is a byte string (see SV.Bytes), as Perl's is, so
// Length, Substr and Index count its bytes.

// byteOrder reads and appends numbers in one byte order. binary.LittleEndian,
// BigEndian and NativeEndian are all one.
type byteOrder interface {
	binary.ByteOrder
	binary.AppendByteOrder
}

// packItem is one item of a template, such as "H*" or "f<3".
type packItem struct {
	letter byte
	order  byteOrder // for f, d and F: < little-endian, > big-endian, else the machine's
	count  int       // how many; for H, h, B and b, how many digits
	star   bool      // * : all of them
	hasN   bool      // a count was given
}

// parseTemplate splits a template into items. Spaces and comments (from # to
// the end of the line) are ignored, as in Perl.
func parseTemplate(op, tmpl string) []packItem {
	var items []packItem
	for i := 0; i < len(tmpl); {
		c := tmpl[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
			continue
		case c == '#':
			for i < len(tmpl) && tmpl[i] != '\n' {
				i++
			}
			continue
		case !strings.ContainsRune("HhBbfdF", rune(c)):
			panic(fmt.Sprintf("scalar: %s template letter %q isn't supported (supported: H h B b f d F)", op, c))
		}
		it := packItem{letter: c, order: binary.NativeEndian}
		i++
		if i < len(tmpl) && (tmpl[i] == '<' || tmpl[i] == '>') {
			if !strings.ContainsRune("fdF", rune(c)) {
				panic(fmt.Sprintf("scalar: %s: %c%c: < and > apply only to f, d and F", op, c, tmpl[i]))
			}
			it.order = binary.LittleEndian
			if tmpl[i] == '>' {
				it.order = binary.BigEndian
			}
			i++
		}
		switch {
		case i < len(tmpl) && tmpl[i] == '*':
			it.star = true
			i++
		case i < len(tmpl) && tmpl[i] >= '0' && tmpl[i] <= '9':
			j := i
			for j < len(tmpl) && tmpl[j] >= '0' && tmpl[j] <= '9' {
				j++
			}
			n, err := strconv.Atoi(tmpl[i:j])
			if err != nil {
				panic(fmt.Sprintf("scalar: %s: bad count %q", op, tmpl[i:j]))
			}
			it.count, it.hasN = n, true
			i = j
		}
		if !it.star && !it.hasN {
			it.count = 1
		}
		items = append(items, it)
	}
	return items
}

// Pack is Perl's pack for templates of H, h, B, b, f, d and F: it packs vals
// (flattened as a list is, see AV.Push) into bytes, returned as a byte string
// (see SV.Bytes).
//
//	scalar.Pack("H*", "4a6f")       // "Jo": hex digits to bytes, high nibble first
//	scalar.Pack("B8", "01001010")   // "J": bits to bytes, highest bit first
//	scalar.Pack("f>", 1.5)          // the 4 bytes of a big-endian float32
//	scalar.Pack("d<*", 1, 2.5)      // two little-endian float64s
//
// H and h take one value, a string of hex digits: H puts the high nibble of
// each byte first, h the low. B and b take a string of bits: B puts the
// highest bit of each byte first, b the lowest. Their count is a number of
// digits or bits (default 1, * for the whole string); a count beyond the
// string pads with zeros, and an odd number of digits or a part-byte of bits
// fills the rest of the last byte with zeros. As in Perl, characters other
// than hex digits, or 0 and 1, aren't an error: Perl's own formula turns
// each into a digit (for bits, the character's lowest bit counts).
//
// f (float32), d and F (float64) take that many values (default 1, * for all
// that are left), each in numeric context; < and > choose little- or
// big-endian, and without either the machine's own order is used, as in
// Perl. A value that's missing counts as undef: "" or 0.
//
// The result is a byte string, so Length, Substr and Index count its bytes.
func Pack(template string, vals ...any) *SV {
	list := flatten(vals)
	next := func() *SV {
		if len(list) == 0 {
			return &SV{}
		}
		v := list[0]
		list = list[1:]
		return v
	}

	var out []byte
	for _, it := range parseTemplate("Pack", template) {
		switch it.letter {
		case 'H', 'h':
			out = packHex(out, next().Str(), it)
		case 'B', 'b':
			out = packBits(out, next().Str(), it)
		case 'f', 'd', 'F':
			n := it.count
			if it.star {
				n = len(list)
			}
			for range n {
				x := next().Num()
				if it.letter == 'f' {
					out = it.order.AppendUint32(out, math.Float32bits(float32(x)))
				} else {
					out = it.order.AppendUint64(out, math.Float64bits(x))
				}
			}
		}
	}
	return &SV{v: out}
}

// hexDigit is the value Perl's pack gives a character as a hex digit: its
// low four bits, after adding 9 to letters, so "a" and "A" are 10, and
// characters that aren't hex digits still give something.
func hexDigit(c byte) byte {
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' {
		return (c + 9) & 0xF
	}
	return c & 0xF
}

// packHex appends the bytes of the hex string s, as pack's H or h does.
func packHex(out []byte, s string, it packItem) []byte {
	n := it.count
	if it.star {
		n = len(s)
	}
	b := make([]byte, (n+1)/2)
	for i := 0; i < n && i < len(s); i++ {
		d := hexDigit(s[i])
		if (i%2 == 0) == (it.letter == 'H') {
			d <<= 4 // H: the high nibble first; h: the low nibble first
		}
		b[i/2] |= d
	}
	return append(out, b...)
}

// packBits appends the bytes of the bit string s, as pack's B or b does.
func packBits(out []byte, s string, it packItem) []byte {
	n := it.count
	if it.star {
		n = len(s)
	}
	b := make([]byte, (n+7)/8)
	for i := 0; i < n && i < len(s); i++ {
		if s[i]&1 == 0 {
			continue
		}
		shift := 7 - i%8 // B: the highest bit first
		if it.letter == 'b' {
			shift = i % 8 // b: the lowest bit first
		}
		b[i/8] |= 1 << shift
	}
	return append(out, b...)
}

// Unpack is Perl's unpack for templates of H, h, B, b, f, d and F: it reads
// data, a string of bytes (a scalar, a Go string or a []byte), and returns
// what the template describes, as an array.
//
//	scalar.Unpack("H*", "Jo")                      // "4a6f"
//	scalar.Unpack("B8", "J")                       // "01001010"
//	scalar.Unpack("f>", scalar.Pack("f>", 1.5))    // 1.5
//	scalar.Unpack("B32", scalar.Pack("f>", 1.5))   // the float's bits: "00111111110000000000000000000000"
//
// H, h, B and b give one string each, of that many hex digits or bits
// (default 1, * for the rest of the data), stopping early if the data runs
// out. f, d and F give that many numbers (default 1, * for as many as the
// rest of the data holds), stopping when there aren't enough bytes left for
// a whole one; bytes left over are ignored, as in Perl. A float32 is widened
// to float64, so 0.1 packed as f comes back as 0.100000001490116, as it does
// in Perl.
func Unpack(template string, data any) *AV {
	var buf []byte
	if b, ok := data.([]byte); ok {
		buf = b
	} else {
		buf = []byte(operand(data).Str())
	}

	out := &AV{}
	pos := 0
	for _, it := range parseTemplate("Unpack", template) {
		rest := buf[pos:]
		switch it.letter {
		case 'H', 'h':
			n := min(it.count, 2*len(rest))
			if it.star {
				n = 2 * len(rest)
			}
			out.items = append(out.items, &SV{v: unpackHex(rest, n, it.letter)})
			pos += (n + 1) / 2
		case 'B', 'b':
			n := min(it.count, 8*len(rest))
			if it.star {
				n = 8 * len(rest)
			}
			out.items = append(out.items, &SV{v: unpackBits(rest, n, it.letter)})
			pos += (n + 7) / 8
		case 'f', 'd', 'F':
			size := 8
			if it.letter == 'f' {
				size = 4
			}
			n := min(it.count, len(rest)/size)
			if it.star {
				n = len(rest) / size
			}
			for i := range n {
				b := rest[i*size : (i+1)*size]
				var x float64
				if size == 4 {
					x = float64(math.Float32frombits(it.order.Uint32(b)))
				} else {
					x = math.Float64frombits(it.order.Uint64(b))
				}
				out.items = append(out.items, &SV{v: x})
			}
			pos += n * size
		}
	}
	return out
}

// unpackHex returns n hex digits of b, as unpack's H or h does.
func unpackHex(b []byte, n int, letter byte) string {
	const digits = "0123456789abcdef"
	var s strings.Builder
	for i := range n {
		d := b[i/2]
		if (i%2 == 0) == (letter == 'H') {
			d >>= 4
		}
		s.WriteByte(digits[d&0xF])
	}
	return s.String()
}

// unpackBits returns n bits of b, as unpack's B or b does.
func unpackBits(b []byte, n int, letter byte) string {
	var s strings.Builder
	for i := range n {
		shift := 7 - i%8
		if letter == 'b' {
			shift = i % 8
		}
		s.WriteByte('0' + b[i/8]>>shift&1)
	}
	return s.String()
}
