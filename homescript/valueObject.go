package homescript

import (
	"fmt"
	"sort"
	"strings"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
	"golang.org/x/exp/maps"
)

// Object value
type ValueObject struct {
	// Can be used if a builtin function only accepts objects of a certain type
	DataType string
	// Specifies whether this object is dynamic
	// If it is dynamic, the analyzer will not run its field checks
	// Such a dynamic object could be the global `ARGS` object
	IsDynamic bool
	// The fields of the object
	ObjFields        map[string]*Value
	CurrentIterIndex *int
	Range            errors.Span
	IsProtected      bool
}

func (self ValueObject) Type() ValueType   { return TypeObject }
func (self ValueObject) Span() errors.Span { return self.Range }
func (self ValueObject) Fields() map[string]*Value {
	self.ObjFields["to_json"] = marshalHelper(self)
	self.ObjFields["to_json_indent"] = marshalIndentHelper(self)
	self.ObjFields["keys"] = valPtr(ValueBuiltinFunction{
		Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			typ := TypeString
			keys := make([]*Value, 0)
			for key, value := range self.ObjFields {
				// Do not include builtin functions in this list
				if (*value).Type() != TypeBuiltinFunction {
					keys = append(keys, valPtr(ValueString{Value: key}))
				}
			}
			return ValueList{ValueType: &typ, Values: &keys}, nil, nil
		},
	})
	return self.ObjFields
}
func (self ValueObject) Index(_ Executor, indexValue Value, span errors.Span) (*Value, bool, *errors.Error) {
	// Check that the indexValue is of type string
	if indexValue.Type() != TypeString {
		return nil, false, errors.NewError(
			span,
			fmt.Sprintf("cannot index value of type '%v' by a value of type '%v'", TypeObject, indexValue.Type()),
			errors.TypeError,
		)
	}
	value, exists := self.ObjFields[indexValue.(ValueString).Value]
	if !exists {
		// Only allow this in dynamic objects
		if self.IsDynamic {
			return nil, false, nil

		}
		return nil, false, errors.NewError(
			span,
			fmt.Sprintf("%v has no member named %s", self.Type(), indexValue.(ValueString).Value),
			errors.TypeError,
		)
	}
	return value, true, nil
}
func (self ValueObject) Protected() bool { return self.IsProtected }
func (self ValueObject) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	fields := make([]string, 0)
	for key, value := range self.ObjFields {
		valueDisplay, err := (*value).Display(executor, span)
		if err != nil {
			return "", err
		}
		fields = append(fields, fmt.Sprintf("%s: %s", key, valueDisplay))
	}
	return fmt.Sprintf("{%s}", strings.Join(fields, "; ")), nil
}
func (self ValueObject) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	fields := make([]string, 0)
	for key, value := range self.ObjFields {
		valueDisplay, err := (*value).Debug(executor, span)
		if err != nil {
			return "", err
		}
		fields = append(fields, fmt.Sprintf("    %s: %s", key, valueDisplay))
	}
	output := "("
	if len(fields) > 0 {
		output += "\n"
	}
	for _, field := range fields {
		output += field + "\n"
	}
	return output + ")", nil
}
func (self ValueObject) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	for _, value := range self.ObjFields {
		valueTrue, err := (*value).IsTrue(executor, span)
		if err != nil {
			return false, err
		}
		if !valueTrue {
			return false, nil
		}
	}
	return true, nil
}
func (self ValueObject) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() == TypeNull {
		return false, nil
	}
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	if len(self.ObjFields) != len(other.(ValueObject).ObjFields) {
		return false, nil
	}
	for key, value := range self.ObjFields {
		eq, err := (*other.(ValueObject).ObjFields[key]).IsEqual(executor, span, *value)
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
	}
	return true, nil
}

func (self *ValueObject) Next() (Value, bool) {
	if self.CurrentIterIndex == nil {
		self.IterReset()
	}

	keys := maps.Keys(self.ObjFields)
	sort.Strings(keys)

	old := *self.CurrentIterIndex
	*self.CurrentIterIndex++

	shouldContinue := *self.CurrentIterIndex <= len(keys)

	if shouldContinue {
		ret := ValuePair{
			Key:         valPtr(makeStr(self.Span(), keys[old])),
			Value:       self.ObjFields[keys[old]],
			IsProtected: true,
		}
		return ret, true
	} else {
		self.IterReset()
		return nil, false
	}
}
func (self *ValueObject) IterReset() {
	zero := 0
	self.CurrentIterIndex = &zero
}
