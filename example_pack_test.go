package scalar_test

// Examples of Pack and Unpack. They're run as tests, and they make up the
// Reference section of README.md (see TestReference).

import (
	"encoding/hex"
	"fmt"

	"github.com/nukopian/scalar"
)

func ExamplePack() {
	fmt.Println(scalar.Pack("H*", "4a6f"))     // hex digits to bytes: "Jo"
	fmt.Println(scalar.Pack("B8", "01001010")) // bits to a byte: "J"

	f := scalar.Pack("f>", 1.5) // a big-endian float32: 4 bytes
	fmt.Println(len(f.Str()), hex.EncodeToString([]byte(f.Str())))
	d := scalar.Pack("d<*", 1, 2.5) // two little-endian float64s
	fmt.Println(len(d.Str()))
	// Output:
	// Jo
	// J
	// 4 3fc00000
	// 16
}

func ExampleUnpack() {
	fmt.Println(scalar.Unpack("H*", "Jo").Get(0))                       // bytes to hex
	fmt.Println(scalar.Unpack("B8", "J").Get(0))                        // a byte to bits
	fmt.Println(scalar.Unpack("H*", scalar.Pack("d>", 3.14159)).Get(0)) // a float64's bytes, in hex
	fmt.Println(scalar.Unpack("B32", scalar.Pack("f>", -2)).Get(0))     // a float32's bits: sign, exponent, fraction
	fmt.Println(scalar.Unpack("f<*", scalar.Pack("f<*", 1.5, 0.1)).Join(", "))
	// Output:
	// 4a6f
	// 01001010
	// 400921f9f01b866e
	// 11000000000000000000000000000000
	// 1.5, 0.100000001490116
}
