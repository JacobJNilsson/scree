package functions

// T holds a value.
type T struct{ n int }

// Value has a value receiver.
func (t T) Value() int { return t.n }

// Pointer has a pointer receiver and a closure.
func (t *T) Pointer() int {
	add := func(d int) int {
		return t.n + d
	}
	return add(1)
}

// Pair is generic.
type Pair[K comparable, V any] struct {
	k K
	v V
}

// Key has a generic pointer receiver.
func (p *Pair[K, V]) Key() K { return p.k }

// Box is generic over one type.
type Box[E any] struct{ e E }

// Get has a generic value receiver.
func (b Box[E]) Get() E { return b.e }

// asm has no body.
func asm() int

var callback = func() int { return 1 }
