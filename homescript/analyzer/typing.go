package analyzer

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Type conversion
//

func (self *Analyzer) ConvertType(oldType pAst.HmsType, createErrors bool) ast.Type {
	switch oldType.Kind() {
	case pAst.NameReferenceParserTypeKind:
		nameType := oldType.(pAst.NameReferenceType)
		switch nameType.Ident.Ident() {
		case "null":
			return ast.NewNullType(oldType.Span())
		case "int":
			return ast.NewIntType(oldType.Span())
		case "float":
			return ast.NewFloatType(oldType.Span())
		case "range":
			return ast.NewRangeType(oldType.Span())
		case "bool":
			return ast.NewBoolType(oldType.Span())
		case "str":
			return ast.NewStringType(oldType.Span())
		default:
			resolved, found := self.currentModule.getType(nameType.Ident.Ident())
			if !found {
				if !createErrors {
					return ast.NewUnknownType()
				}

				self.error(
					fmt.Sprintf("Illegal use of undeclared type '%s'", nameType.Ident.Ident()),
					[]string{fmt.Sprintf("Consider declaring the type like this: `type %s = ...`", nameType.Ident.Ident())},
					nameType.Span(),
				)

				return ast.NewUnknownType()
			}
			// mark the resolved type as `used`
			resolved.Used = true
			return resolved.Type
		}
	case pAst.ObjectFieldsParserTypeKind:
		objType := oldType.(pAst.ObjectType)

		switch objType.Fields.Kind() {
		case pAst.AnyObjectTypeFieldTypeKind:
			return ast.NewAnyObjectType(objType.Range)
		case pAst.NormalObjectTypeFieldTypeKind:
			fields := objType.Fields.(pAst.ObjectTypeFieldTypeFields).Fields
			newFields := make([]ast.ObjectTypeField, 0)

			// check that no object fields double
			for _, field := range fields {
				for _, toCheck := range newFields {
					if toCheck.FieldName.Ident() == field.FieldName.Ident() {
						if !createErrors {
							return ast.NewUnknownType()
						}

						self.error(
							fmt.Sprintf("Object type field '%s' is declared twice", field.FieldName),
							nil,
							field.FieldName.Span(),
						)

						return ast.NewUnknownType()
					}
				}

				newFields = append(newFields, ast.NewObjectTypeField(
					field.FieldName,
					self.ConvertType(field.Type, createErrors),
					field.Range,
				))
			}

			return ast.NewObjectType(newFields, objType.Range)
		default:
			panic("A new field kind was introduced without updating this code")
		}
	case pAst.ListTypeKind:
		listType := oldType.(pAst.ListType)
		return ast.NewListType(self.ConvertType(listType.Inner, createErrors), oldType.Span())
	case pAst.FunctionTypeKind:
		fnType := oldType.(pAst.FunctionType)
		newParams := make([]ast.FunctionTypeParam, 0)

		// check that there are no duplicate params
		for _, param := range fnType.Params {
			for _, toCheck := range newParams {
				if toCheck.Name.Ident() == param.Name.Ident() {
					if !createErrors {
						return ast.NewUnknownType()
					}

					self.error(
						fmt.Sprintf("Duplicate parameter name '%s' in type declaration", param.Name),
						nil,
						param.Name.Span(),
					)

					return ast.NewUnknownType()
				}

				newParams = append(newParams, ast.NewFunctionTypeParam(param.Name, self.ConvertType(param.Type, createErrors)))
			}
		}

		return ast.NewFunctionType(
			ast.NewNormalFunctionTypeParamKind(newParams),
			fnType.ParamsSpan,
			self.ConvertType(fnType.ReturnType, createErrors),
			oldType.Span(),
		)
	case pAst.OptionParserTypeKind:
		optionType := oldType.(pAst.OptionType)
		return ast.NewOptionType(
			self.ConvertType(optionType.Inner, true),
			oldType.Span(),
		)
	default:
		panic(fmt.Sprintf("A new type kind ('%v') was introduced without updating this code", oldType.Kind()))
	}
}

//
// Type compatibility
//

type CompabilityError struct {
	GotDiagnostic      diagnostic.Diagnostic
	ExpectedDiagnostic *diagnostic.Diagnostic
}

func newCompabilityErr(lhs diagnostic.Diagnostic, rhs *diagnostic.Diagnostic) *CompabilityError {
	return &CompabilityError{
		GotDiagnostic:      lhs,
		ExpectedDiagnostic: rhs,
	}
}

// returns a true if an any type is detected
func (self *Analyzer) CheckAny(typ ast.Type) bool {
	switch typ.Kind() {
	case ast.AnyTypeKind:
		return true
	case ast.UnknownTypeKind, ast.NeverTypeKind,
		ast.NullTypeKind, ast.IntTypeKind,
		ast.FloatTypeKind, ast.BoolTypeKind,
		ast.StringTypeKind, ast.RangeTypeKind,
		ast.AnyObjectTypeKind:
		return false
	case ast.ListTypeKind:
		listType := typ.(ast.ListType)
		return self.CheckAny(listType.Inner)
	case ast.ObjectTypeKind:
		objType := typ.(ast.ObjectType)
		for _, field := range objType.ObjFields {
			if self.CheckAny(field.Type) {
				return true
			}
		}
		return false
	case ast.OptionTypeKind:
		optType := typ.(ast.OptionType)
		return self.CheckAny(optType.Inner)
	case ast.FnTypeKind:
		fnType := typ.(ast.FunctionType)

		switch fnType.Params.Kind() {
		case ast.NormalFunctionTypeParamKindIdentifierKind:
			normalKind := fnType.Params.(ast.NormalFunctionTypeParamKindIdentifier)
			for _, param := range normalKind.Params {
				if self.CheckAny(param.Type) {
					return true
				}
			}
			return self.CheckAny(fnType.ReturnType)
		case ast.VarArgsFunctionTypeParamKindIdentifierKind:
			varKind := fnType.Params.(ast.VarArgsFunctionTypeParamKindIdentifier)
			for _, typ := range varKind.ParamTypes {
				if self.CheckAny(typ) {
					return true
				}
			}
			return self.CheckAny(varKind.RemainingType)
		default:
			panic("A new param kind was introduced without updating this code")
		}
	default:
		panic(fmt.Sprintf("`CheckAny` was called on an unsupported type: '%s'", typ))
	}
}

func (self *Analyzer) TypeCheck(got ast.Type, expected ast.Type, allowFunctionTypes bool) *CompabilityError {
	// allow the `any` type if it is expected
	switch expected.Kind() {
	case ast.AnyTypeKind, ast.UnknownTypeKind, ast.NeverTypeKind:
		return nil
	}

	switch got.Kind() {
	case ast.UnknownTypeKind, ast.NeverTypeKind:
		return nil
	case ast.AnyTypeKind:
		// NOTE: this is OK since the `any` type is handled elsewhere
		return nil
	case ast.NullTypeKind:
		err, _ := self.checkTypeKindEquality(got, expected)
		return err
	case ast.IntTypeKind, ast.FloatTypeKind,
		ast.BoolTypeKind, ast.StringTypeKind,
		ast.RangeTypeKind:

		err, _ := self.checkTypeKindEquality(got, expected)
		return err
	case ast.ListTypeKind:
		err, proceed := self.checkTypeKindEquality(got, expected)
		if err != nil || !proceed {
			return err
		}

		lhsType := got.(ast.ListType)
		rhsType := expected.(ast.ListType)

		// check inner type
		if err := self.TypeCheck(lhsType.Inner, rhsType.Inner, allowFunctionTypes); err != nil {
			return err
		}
	case ast.AnyObjectTypeKind:
		err, proceed := self.checkTypeKindEquality(got, expected)
		if err != nil || !proceed {
			return err
		}
		return nil
	case ast.ObjectTypeKind:
		gotObj := got.(ast.ObjectType)

		err, proceed := self.checkTypeKindEquality(got, expected)
		if err != nil || !proceed {
			return err
		}
		expectedObj := expected.(ast.ObjectType)

		// check if all expected fields exist on the `got` object
		// if a field on does not exist, return an error
		for _, expectedField := range expectedObj.ObjFields {
			var gotField *ast.ObjectTypeField = nil
			for _, otherField := range gotObj.ObjFields {
				if otherField.FieldName.Ident() == expectedField.FieldName.Ident() {
					gotField = &otherField
					break
				}
			}

			// the expected field was not found on the `got` object
			if gotField == nil {
				span := gotObj.Span()
				if len(gotObj.ObjFields) > 0 {
					span = gotObj.ObjFields[len(gotObj.ObjFields)-1].Span
				}

				return newCompabilityErr(
					diagnostic.Diagnostic{
						Level:   diagnostic.DiagnosticLevelError,
						Message: fmt.Sprintf("Field '%s' is missing", expectedField.FieldName.Ident()),
						Notes:   nil,
						Span:    span,
					},
					&diagnostic.Diagnostic{
						Level:   diagnostic.DiagnosticLevelHint,
						Message: "Field expected due to this",
						Notes:   nil,
						Span:    expectedField.FieldName.Span(),
					},
				)
			}

			// check field type equality
			if err := self.TypeCheck(gotField.Type, expectedField.Type, allowFunctionTypes); err != nil {
				return err
			}
		}

		// check if the `got` object holds any excess keys
		for _, gotField := range gotObj.ObjFields {
			// if this field does not exist on the `expected` object, cause an error
			found := false
			for _, expectedField := range expectedObj.ObjFields {
				if gotField.FieldName.Ident() == expectedField.FieldName.Ident() {
					found = true
					break
				}
			}

			// this field does not exist on the `expected` object
			if !found {
				return newCompabilityErr(
					diagnostic.Diagnostic{
						Level:   diagnostic.DiagnosticLevelError,
						Message: fmt.Sprintf("Found unexpected field '%s'", gotField.FieldName.Ident()),
						Notes:   nil,
						Span:    gotField.FieldName.Span(),
					},
					&diagnostic.Diagnostic{
						Level:   diagnostic.DiagnosticLevelHint,
						Message: fmt.Sprintf("Field '%s' does not exist on this type", gotField.FieldName.Ident()),
						Notes:   nil,
						Span:    expected.Span(),
					},
				)
			}
		}
	case ast.OptionTypeKind:
		gotOpt := got.(ast.OptionType)
		err, proceed := self.checkTypeKindEquality(got, expected)
		if err != nil || !proceed {
			return err
		}

		if expected.Kind() == ast.OptionTypeKind {
			expectedOpt := expected.(ast.OptionType)
			return self.TypeCheck(gotOpt.Inner, expectedOpt.Inner, true)
		}

		return nil
	case ast.FnTypeKind:
		if !allowFunctionTypes {
			return newCompabilityErr(
				diagnostic.Diagnostic{
					Level:   diagnostic.DiagnosticLevelError,
					Message: "Cannot cast a function value at runtime",
					Notes:   nil,
					Span:    expected.Span(),
				},
				&diagnostic.Diagnostic{
					Level:   diagnostic.DiagnosticLevelHint,
					Message: "Possible function value found here",
					Notes:   []string{"If this function is called later, cast its return value: `func() as type`"},
					Span:    got.Span(),
				},
			)
		}

		err, proceed := self.checkTypeKindEquality(got, expected)
		if err != nil || !proceed {
			return err
		}
		gotFn := got.(ast.FunctionType)
		expectedFn := expected.(ast.FunctionType)
		// check return type
		if err := self.TypeCheck(gotFn.ReturnType, expectedFn.ReturnType, allowFunctionTypes); err != nil {
			// TODO: include better error message
			return err
		}
		if expectedFn.Params.Kind() != gotFn.Params.Kind() {
			return newCompabilityErr(
				diagnostic.Diagnostic{
					Level:   diagnostic.DiagnosticLevelError,
					Message: fmt.Sprintf("Expected parameter kind '%s', found '%s'", expectedFn.Params.Kind(), gotFn.Params.Kind()),
					Notes:   []string{"There is a difference between a function which takes a fixed number of arguments and one which can take an arbitrary amount"},
					Span:    gotFn.ParamsSpan,
				},
				nil,
			)
		}
		switch expectedFn.Params.Kind() {
		case ast.NormalFunctionTypeParamKindIdentifierKind:
			// check that all parameters of the `expected` function exist on the `got` function
			expectedFnParams := expectedFn.Params.(ast.NormalFunctionTypeParamKindIdentifier)
			gotFnParams := gotFn.Params.(ast.NormalFunctionTypeParamKindIdentifier)
			for _, expectedParam := range expectedFnParams.Params {
				var foundParam *ast.FunctionTypeParam = nil
				for _, gotParam := range gotFnParams.Params {
					if expectedParam.Name.Ident() == gotParam.Name.Ident() {
						foundParam = &gotParam
						break
					}
				}
				if foundParam == nil {
					return newCompabilityErr(
						diagnostic.Diagnostic{
							Level:   diagnostic.DiagnosticLevelError,
							Message: fmt.Sprintf("Parameter '%s' is missing", expectedParam.Name.Ident()),
							Notes:   nil,
							Span:    gotFn.ParamsSpan,
						},
						&diagnostic.Diagnostic{
							Level:   diagnostic.DiagnosticLevelHint,
							Message: "Parameter expected due to this",
							Notes:   nil,
							Span:    expectedParam.Name.Span(),
						},
					)
				}
				// check type equality of the param type
				if err := self.TypeCheck(foundParam.Type, expectedParam.Type, allowFunctionTypes); err != nil {
					return err
				}
			}
		case ast.VarArgsFunctionTypeParamKindIdentifierKind:
		default:
			panic("A new function parameter type kind was introduced without updating this code")
		}
	default:
		panic("A new type kind was added without updating this code")
	}
	return nil
}

func (self *Analyzer) checkTypeKindEquality(got ast.Type, expected ast.Type) (err *CompabilityError, proceed bool) {
	if got.Kind() == ast.UnknownTypeKind || got.Kind() == ast.NeverTypeKind ||
		expected.Kind() == ast.UnknownTypeKind || expected.Kind() == ast.NeverTypeKind {
		return nil, false
	}

	if expected.Kind() != got.Kind() {
		return newCompabilityErr(
			diagnostic.Diagnostic{
				Level:   diagnostic.DiagnosticLevelError,
				Message: fmt.Sprintf("Mismatched types: expected '%s', got '%s'", expected.Kind(), got.Kind()),
				Notes:   nil,
				Span:    got.Span(),
			},
			&diagnostic.Diagnostic{
				Level:   diagnostic.DiagnosticLevelHint,
				Message: fmt.Sprintf("Type '%s' expected due to this", expected.Kind()),
				Notes:   nil,
				Span:    expected.Span(),
			},
		), false
	}
	return nil, true
}
