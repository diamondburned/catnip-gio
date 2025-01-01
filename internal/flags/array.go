package flags

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/pflag"
)

// Array is a pflag.Value that holds an array of values.
type Array[T pflag.Value] struct {
	Separator string
	Values    []T
	IsSet     bool
}

var _ pflag.Value = (*Array[*ColorNRGBA])(nil)

func NewArray[T pflag.Value](separator string, values ...T) *Array[T] {
	return &Array[T]{
		Separator: separator,
		Values:    values,
		IsSet:     false,
	}
}

func (a *Array[T]) Set(s string) error {
	if !a.IsSet {
		a.Values = nil
		a.IsSet = true
	}

	for _, s := range strings.Split(s, a.Separator) {
		s = strings.TrimSpace(s)

		v := reflect.New(reflect.TypeFor[T]().Elem()).Interface().(pflag.Value)
		if err := v.Set(s); err != nil {
			return fmt.Errorf("invalid value %q: %w", s, err)
		}

		a.Values = append(a.Values, v.(T))
	}

	return nil
}

func (a *Array[T]) At(i int) T {
	return a.Values[i]
}

func (a *Array[T]) String() string {
	values := make([]string, len(a.Values))
	for i, v := range a.Values {
		values[i] = v.String()
	}
	return strings.Join(values, a.Separator)
}

func (a *Array[T]) Type() string {
	return "array"
}
