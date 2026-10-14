package scalar

// To converts v into a Go value of type T, the type in brackets: the way
// out of scalar into ordinary Go, once you know the shape you want.
//
//	skills, err := scalar.To[[]string](doc.At("skills"))
//	salary, err := scalar.To[int](doc.At("salary.max"))
//	posting, err := scalar.To[Posting](doc)
//	counts, err := scalar.To[map[string]int](bag)
//
// It converts as Decode does, so the rules are Decode's: a hash fills a
// struct or a map, an array fills a slice, numbers and strings convert as
// Perl converts them but fail where Perl would warn, and the error, a
// *DecodeError, says where the value is, as a path for At. On an error, To
// returns T's zero value.
//
// v may be a scalar, an array, a hash, or any Go value Scalar accepts. A
// set gives its members, as an array, and a bag its counts, as a hash, so
// To[[]string](set) and To[map[string]int](bag) work.
//
// Unlike StrSlice, IntSlice and NumSlice, which convert every element as
// Perl's string or numeric context does and never fail, To checks: "abc"
// isn't an int. Unlike ValuesSlice and PairsMap, which give the stored
// scalars themselves, To gives copies, which share nothing with v.
//
// To is a function rather than a method because Go's methods can't take a
// type in brackets.
func To[T any](v any) (T, error) {
	var out T
	if err := toScalar(v).Decode(&out); err != nil {
		var zero T
		return zero, err
	}
	return out, nil
}

// toScalar is v as a scalar for To: arrays and hashes as references to
// them, a set's members, a bag's counts, and anything else as Scalar has it.
func toScalar(v any) *SV {
	switch x := v.(type) {
	case *SV:
		if x == nil {
			return &SV{}
		}
		return x
	case *AV:
		if x == nil {
			return &SV{}
		}
		return &SV{v: x}
	case *HV:
		if x == nil {
			return &SV{}
		}
		return &SV{v: x}
	case *Set:
		return &SV{v: x.Members()}
	case *Bag:
		return &SV{v: x.Counts()}
	}
	return Scalar(v)
}
