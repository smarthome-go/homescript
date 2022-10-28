package homescript

import (
	"fmt"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

// Pair value
type ValuePair struct {
	Key         *Value
	Value       *Value
	Range       errors.Span
	IsProtected bool
}

func (self ValuePair) Type() ValueType   { return TypePair }
func (self ValuePair) Span() errors.Span { return self.Range }
func (self ValuePair) Fields() map[string]*Value {
	return map[string]*Value{
		"k":              self.Key,
		"v":              self.Value,
		"to_json":        marshalHelper(self),
		"to_json_indent": marshalIndentHelper(self),
	}
}
func (self ValuePair) Index(_ Executor, _ Value, span errors.Span) (*Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValuePair) Protected() bool { return self.IsProtected }
func (self ValuePair) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	key, err := (*self.Key).Display(executor, span)
	if err != nil {
		return "", nil
	}
	value, err := (*self.Value).Display(executor, span)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s => %s", key, value), nil
}
func (self ValuePair) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	key, err := (*self.Key).Debug(executor, span)
	if err != nil {
		return "", nil
	}
	value, err := (*self.Value).Debug(executor, span)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(Key: %s | Value: %s)", key, value), nil
}
func (self ValuePair) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	key, err := (*self.Key).IsTrue(executor, span)
	if err != nil {
		return false, err
	}
	value, err := (*self.Value).IsTrue(executor, span)
	if err != nil {
		return false, err
	}
	return key && value, nil
}
func (self ValuePair) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	key, err := (*self.Key).IsEqual(executor, span, (*other.(ValuePair).Key))
	if err != nil {
		return false, err
	}
	value, err := (*self.Value).IsEqual(executor, span, (*other.(ValuePair).Value))
	if err != nil {
		return false, err
	}
	return key && value, nil
}
