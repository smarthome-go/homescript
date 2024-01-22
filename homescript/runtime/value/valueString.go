package value

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueString struct {
	Inner       string
	currIterIdx *int
}

func (_ ValueString) Kind() ValueKind { return StringValueKind }

func (self ValueString) Display() (string, *VmInterrupt) { return self.Inner, nil }

func (self ValueString) IsEqual(other Value) (bool, *VmInterrupt) {
	return self.Inner == other.(ValueString).Inner, nil
}

func (self ValueString) Fields() (map[string]*Value, *VmInterrupt) {
	return map[string]*Value{
		"len": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			return NewValueInt(int64(utf8.RuneCountInString(self.Inner))), nil
		}),
		"replace": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			replace := args[0].(ValueString).Inner
			replaceWith := args[1].(ValueString).Inner
			out := strings.ReplaceAll(self.Inner, replace, replaceWith)
			return NewValueString(out), nil
		}),
		"repeat": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			count := int(args[0].(ValueInt).Inner)
			return NewValueString(strings.Repeat(self.Inner, count)), nil
		}),
		"split": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			sep := args[0].(ValueString).Inner
			list := strings.Split(self.Inner, sep)
			valueList := make([]*Value, 0)
			for _, item := range list {
				valueList = append(valueList, NewValueString(item))
			}
			return NewValueList(valueList), nil
		}),
		"contains": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			test := args[0].(ValueString).Inner
			return NewValueBool(strings.Contains(self.Inner, test)), nil
		}),
		"to_lower": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			return NewValueString(strings.ToLower(self.Inner)), nil
		}),
		"to_upper": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			return NewValueString(strings.ToUpper(self.Inner)), nil
		}),
		"parse_int": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			res, err := strconv.ParseInt(self.Inner, 10, 64)
			if err != nil {
				return nil, NewVMThrowInterrupt(span, err.Error())
			}
			return NewValueInt(res), nil
		}),
		"parse_float": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			res, err := strconv.ParseFloat(self.Inner, 64)
			if err != nil {
				return nil, NewVMThrowInterrupt(span, err.Error())
			}
			return NewValueFloat(res), nil
		}),
		"parse_bool": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			res, err := strconv.ParseBool(self.Inner)
			if err != nil {
				return nil, NewVMThrowInterrupt(span, err.Error())
			}
			return NewValueBool(res), nil
		}),
		"parse_json": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			var raw interface{}
			if err := json.Unmarshal([]byte(self.Inner), &raw); err != nil {
				return nil, NewVMThrowInterrupt(span, fmt.Sprintf("JSON parse error: %s", err.Error()))
			}
			value, i := UnmarshalValue(span, raw)
			if i != nil {
				return nil, i
			}

			return value, nil
		}),
	}, nil
}

func (self ValueString) iterNext() (Value, bool) {
	if self.currIterIdx == nil {
		self.iterReset()
	}

	old := *self.currIterIdx
	*self.currIterIdx++

	shouldContinue := *self.currIterIdx <= len(self.Inner)

	if shouldContinue {
		return *NewValueString(
			fmt.Sprint(self.Inner[old]),
		), true
	} else {
		self.iterReset()
		return nil, false
	}
}

func (self ValueString) iterReset() {
	*self.currIterIdx = 0
}

func (self ValueString) IntoIter() func() (Value, bool) {
	return self.iterNext
}

func NewValueString(inner string) *Value {
	zero := 0
	val := Value(ValueString{Inner: inner, currIterIdx: &zero})
	return &val
}
