package homescript

import (
	"fmt"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

// Enum value
type ValueEnum struct {
	Variants         []ValueEnumVariant
	CurrentIterIndex *int
	Range            errors.Span
	IsProtected      bool
}

func (self ValueEnum) Type() ValueType   { return TypeEnum }
func (self ValueEnum) Span() errors.Span { return self.Range }
func (self ValueEnum) Fields() map[string]*Value {
	panic("`Fields` is not called on a bare enum")
}

func (self ValueEnum) Index(_ Executor, indexValue Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueEnum) Protected() bool { return true }
func (self ValueEnum) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	panic("`Display` is not called on a bare enum")
}
func (self ValueEnum) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	panic("`Debug` is not called on a bare enum")
}
func (self ValueEnum) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	panic("`IsTrue` is not called on a bare enum")
}
func (self ValueEnum) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	panic("`IsEqual` is not called on a bare enum")
}

func (self ValueEnum) HasVariant(toCheck string) bool {
	for _, variant := range self.Variants {
		if variant.Name == toCheck {
			return true
		}
	}
	return false
}

// EnumVariant value
type ValueEnumVariant struct {
	Name  string
	Range errors.Span
}

func (self ValueEnumVariant) Type() ValueType   { return TypeEnumVariant }
func (self ValueEnumVariant) Span() errors.Span { return self.Range }
func (self ValueEnumVariant) Fields() map[string]*Value {
	panic("`Fields` is not called on an enum variant")
}

func (self ValueEnumVariant) Index(_ Executor, indexValue Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueEnumVariant) Protected() bool { return false }
func (self ValueEnumVariant) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	return self.Name, nil
}
func (self ValueEnumVariant) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	return self.Display(executor, span)
}
func (self ValueEnumVariant) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	return true, nil
}
func (self ValueEnumVariant) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return self.Name == other.(ValueEnumVariant).Name, nil
}
