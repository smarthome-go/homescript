package value

import (
	"context"
	"fmt"
	"sort"
	"strings"

	errors "github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueObject struct {
	FieldsInternal map[string]*Value
}

func (_ ValueObject) Kind() ValueKind { return ObjectValueKind }

func (self ValueObject) Display() (string, *VmInterrupt) {
	fields := make([]string, 0)
	for key, field := range self.FieldsInternal {
		disp, err := (*field).Display()
		if err != nil {
			return "", err
		}
		disp = strings.ReplaceAll(disp, "\n", "\n    ")
		fields = append(fields, fmt.Sprintf("%s: %s", key, disp))
	}

	return fmt.Sprintf("{\n    %s\n}", strings.Join(fields, ",\n    ")), nil
}

func (self ValueObject) IsEqual(other Value) (bool, *VmInterrupt) {
	otherObj := other.(ValueObject)

	for key, value := range self.FieldsInternal {
		otherValue, found := otherObj.FieldsInternal[key]
		if !found {
			return false, nil
		}
		isEqual, i := (*value).IsEqual(*otherValue)
		if i != nil {
			return false, i
		}
		if !isEqual {
			return false, nil
		}
	}

	return true, nil
}

func (self ValueObject) Fields() (map[string]*Value, *VmInterrupt) {
	fields := map[string]*Value{
		"to_string": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			display, i := self.Display()
			if i != nil {
				return nil, i
			}
			return NewValueString(display), nil
		}),
		"keys": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			rawKeys := make([]string, 0)
			for key := range self.FieldsInternal {
				rawKeys = append(rawKeys, key)
			}
			sort.Strings(rawKeys)

			keys := make([]*Value, 0)
			for _, key := range rawKeys {
				keys = append(keys, NewValueString(key))
			}

			return NewValueList(keys), nil
		}),
		"to_json":        MarshalToString(self),
		"to_json_indent": MarshalIndentToString(self),
	}

	for key, val := range self.FieldsInternal {
		fields[key] = val
	}

	return fields, nil
}

func (self ValueObject) IntoIter() func() (Value, bool) {
	panic("A value of type object cannot be used as an iterator")
}

func NewValueObject(fields map[string]*Value) *Value {
	val := Value(ValueObject{
		FieldsInternal: fields,
	})
	return &val
}

func (self ValueObject) IntoAnyObject() *Value {
	return NewValueAnyObject(self.FieldsInternal)
}
