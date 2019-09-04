package postgres

import "strconv"

// placeholderMap is used to map placeholders ($n) to field values.
//
// The index of the slice is used as placeholder position,
// while the value of the slice is used as a reference to the field's position.
type placeholderMap []int

func newPlaceholderMap() *placeholderMap {
	p := make(placeholderMap, 0)
	return &p
}

// assign assigns placeholders to given fields and returns
// a slice of placeholders [$n, $n+1, ...]
func (p *placeholderMap) assign(f ...*field) []string {
	out := make([]string, 0)
	for _, x := range f {
		out = append(out, p.next(x))
	}
	return out
}

// next assigns the next placeholder to a given field
// and returns the placeholder $n
func (p *placeholderMap) next(f *field) string {
	for placeholder, position := range *p {
		if position == f.position {
			return "$" + strconv.Itoa(placeholder+1)
		}
	}

	*p = append(*p, f.position)
	return "$" + strconv.Itoa(len(*p))
}

// args returns the field's values in order of the placeholder positions
func (p *placeholderMap) args(fs fields) []interface{} {
	out := make([]interface{}, 0)
	for _, position := range *p {
		for _, f := range fs {
			if f.position == position {
				out = append(out, f)
				break
			}
		}
	}
	return out
}
