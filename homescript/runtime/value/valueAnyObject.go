package value

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueAnyObject struct {
	FieldsInternal map[string]*Value
}

func (_ ValueAnyObject) Kind() ValueKind { return AnyObjectValueKind }

func (self ValueAnyObject) Display() (string, *VmInterrupt) {
	fields := make([]string, 0)
	for key, field := range self.FieldsInternal {
		disp, err := (*field).Display()
		if err != nil {
			return "", err
		}
		fields = append(fields, fmt.Sprintf("%s: %s", key, disp))
	}

	return fmt.Sprintf("{\n    %s\n}", strings.Join(fields, ",\n    ")), nil
}

func (self ValueAnyObject) IsEqual(other Value) (bool, *VmInterrupt) {
	otherObj := other.(ValueAnyObject)

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

func (self ValueAnyObject) Fields() (map[string]*Value, *VmInterrupt) {
	return map[string]*Value{
		"set": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			self.FieldsInternal[args[0].(ValueString).Inner] = &args[1]
			return NewValueNull(), nil
		}),
		"get": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			value := self.FieldsInternal[args[0].(ValueString).Inner]
			return NewValueOption(value), nil
		}),
		"get_type": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			value := self.FieldsInternal[args[0].(ValueString).Inner]
			return NewValueString((*value).Kind().TypeKind().String()), nil
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
		"to_string": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			disp, i := self.Display()
			if i != nil {
				return nil, i
			}
			return NewValueString(disp), nil
		}),
		"to_json":        MarshalToString(self),
		"to_json_indent": MarshalIndentToString(self),
	}, nil
}

func (self ValueAnyObject) IntoIter() func() (Value, bool) {
	panic("A value of type object cannot be used as an iterator")
}

func (self ValueAnyObject) Clone() *Value {
	newFields := make(map[string]*Value)

	for key, value := range self.FieldsInternal {
		newFields[key] = (*value).Clone()
	}

	return NewValueAnyObject(newFields)
}

func NewValueAnyObject(fields map[string]*Value) *Value {
	val := Value(ValueAnyObject{
		FieldsInternal: fields,
	})
	return &val
}
