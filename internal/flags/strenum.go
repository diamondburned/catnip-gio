package flags

import "fmt"

type StringEnum[T ~string] struct {
	Value   T
	Allowed []T
}

func NewStringEnum[T ~string](allowed ...T) *StringEnum[T] {
	return &StringEnum[T]{
		Value:   allowed[0],
		Allowed: allowed,
	}
}

func (e *StringEnum[T]) Set(s string) error {
	for _, v := range e.Allowed {
		if v == T(s) {
			e.Value = T(s)
			return nil
		}
	}
	return fmt.Errorf("invalid value %q", s)
}

func (e *StringEnum[T]) String() string {
	return fmt.Sprintf("%v", e.Value)
}

func (e *StringEnum[T]) Type() string {
	return "strenum"
}
