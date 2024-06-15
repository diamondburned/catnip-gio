package flags

import (
	"fmt"
	"reflect"

	"github.com/spf13/pflag"
)

// Array is a pflag.Value that holds an array of values.
type Array[T pflag.Value] struct {
	Values []T
	IsSet  bool
}

var _ pflag.Value = (*Array[*ColorNRGBA])(nil)

func NewArray[T pflag.Value](values ...T) *Array[T] {
	return &Array[T]{
		Values: values,
		IsSet:  false,
	}
}

func (a *Array[T]) Set(s string) error {
	if !a.IsSet {
		a.Values = nil
		a.IsSet = true
	}

	v := reflect.New(reflect.TypeFor[T]().Elem()).Interface().(pflag.Value)
	if err := v.Set(s); err != nil {
		return err
	}

	a.Values = append(a.Values, v.(T))
	return nil
}

func (a *Array[T]) At(i int) T {
	return a.Values[i]
}

func (a *Array[T]) String() string {
	return fmt.Sprintf("%v", *a)
}

func (a *Array[T]) Type() string {
	return "array"
}
