package analyzer

import (
	"errors"
	"go/types"
	"log"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

func unTuple(typ types.Type) [2]types.Type {
	if typ == nil {
		return [2]types.Type{types.Typ[types.Invalid], typeError.Type()}
	}

	tuple, ok := typ.(*types.Tuple)
	if !ok {
		return [2]types.Type{typ, nil}
	}

	if tuple.Len() == 0 {
		return [2]types.Type{types.Typ[types.Invalid], typeError.Type()}
	} else if tuple.Len() == 1 {
		return [2]types.Type{tuple.At(0).Type(), nil}
	} else if tuple.Len() == 2 {
		return [2]types.Type{tuple.At(0).Type(), tuple.At(1).Type()}
	}

	return [2]types.Type{types.Typ[types.Invalid], typeError.Type()}
}

var errTemplateUndefined error = errors.New("template undefined")
var errEmptyExpression error = errors.New("empty expression")

var errArgumentsOnlyForFunction error = errors.New(
	"only functions and methods accept arguments",
)
var errFunctionUndefined error = errors.New("function undefined")

var errFunctionParameterSizeMismatch error = errors.New(
	"function 'parameter' and 'argument' size mismatch",
)
var errFunctionNotEnoughArguments error = errors.New("not enough arguments for function")
var errFunctionMaxReturn error = errors.New("function cannot return more than 2 values")
var errFunctionVoidReturn error = errors.New("function cannot have 'void' return value")

var errFunctionSecondReturnNotError error = errors.New(
	"function's second return value must be of type 'error'",
)
var errTypeMismatch error = errors.New("type mismatch")
var errVariableNotUsed error = errors.New("variable is never used")
var errDefeatedTypeSystem error = errors.New("type system defeated")

func makeExpressionTypeCheck(
	symbols []*lexer.Token,
	typs []types.Type,
	makeTypeInference InferenceFunc,
	nodeRange lexer.Range,
) (resultType [2]types.Type, variablesToRecheck []*collectionPostCheckImplicitTypeNode, errs []*parser.ParseError) {
	if len(symbols) != len(typs) {
		log.Printf("every symbol must have a single type."+
			"\n symbols = %q\n typs = %q", symbols, typs)
		panic("every symbol must have a single type")
	}

	// 1. len(symbols) == 0 && len == 1
	if len(symbols) == 0 {
		err := parser.NewParseError(&lexer.Token{}, errEmptyExpression)
		err.Range = nodeRange
		errs = append(errs, err)

		return unTuple(types.Typ[types.Invalid]), nil, errs
	} else if len(symbols) == 1 {
		symbol := symbols[0]
		typ := typs[0]

		funcType, ok := typ.(*types.Signature)
		if ok {
			returnType, _, localErrs := makeFunctionTypeCheck(
				funcType,
				symbol,
				typs[1:],
				symbols[1:],
				makeTypeInference,
			)
			errs = append(errs, localErrs...)

			return unTuple(returnType), nil, errs
		}

		return unTuple(typ), nil, nil
	}

	// 2. len(symbols) >= 2 :: Always true if this section is reached
	funcType, ok := typs[0].(*types.Signature)
	if !ok {
		// If the type is 'any', it could be a method call at runtime.
		// Don't error on method calls with arguments when the receiver type is unknown.
		// This prevents false positives like `.Format "2006-01-02"` when the type of
		// the receiver is inferred as 'any' due to lack of type information.
		if types.Identical(typs[0], typeAny.Type()) {
			return unTuple(typeAny.Type()), nil, nil
		}

		err := parser.NewParseError(symbols[0], errArgumentsOnlyForFunction)
		errs = append(errs, err)

		return unTuple(types.Typ[types.Invalid]), nil, errs
	}

	returnType, rechecks, localErrs := makeFunctionTypeCheck(
		funcType,
		symbols[0],
		typs[1:],
		symbols[1:],
		makeTypeInference,
	)

	errs = append(errs, localErrs...)
	variablesToRecheck = append(variablesToRecheck, rechecks...)

	return unTuple(returnType), variablesToRecheck, errs
}

func makeFunctionTypeCheck(
	funcType *types.Signature,
	funcSymbol *lexer.Token,
	argTypes []types.Type,
	argSymbols []*lexer.Token,
	makeTypeInference InferenceFunc,
) (resultType types.Type, variablesToRecheck []*collectionPostCheckImplicitTypeNode, errs []*parser.ParseError) {
	if funcType == nil {
		err := parser.NewParseError(funcSymbol, errFunctionUndefined)
		errs = append(errs, err)
		return types.Typ[types.Invalid], nil, errs
	}

	// 1. Check Parameter VS Argument validity
	invalidReturnType := types.Typ[types.Invalid]

	paramSize := funcType.Params().Len()
	argumentSize := len(argSymbols)
	isVariadicFunction := funcType.Variadic()

	if !isVariadicFunction && paramSize != argumentSize {
		err := parser.NewParseError(funcSymbol, errFunctionParameterSizeMismatch)
		errs = append(errs, err)
		return invalidReturnType, nil, errs
	} else if isVariadicFunction && argumentSize < paramSize-1 {
		// For variadic functions, minimum args is paramSize-1 (variadic param is optional)
		err := parser.NewParseError(funcSymbol, errFunctionNotEnoughArguments)
		errs = append(errs, err)
		return invalidReturnType, nil, errs
	}

	lastParamIndex := paramSize - 1
	var paramType, argumentType types.Type

	for i := range argSymbols {
		if isVariadicFunction && i >= lastParamIndex {
			sliceParam, ok := funcType.Params().At(lastParamIndex).Type().(*types.Slice)
			if !ok {
				panic(
					"expected variadic function with last param being a slice but didn't find the slice",
				)
			}
			paramType = sliceParam.Elem()

			// Check if argument is a slice that can be expanded to variadic
			// In Go templates, passing []T to ...T is valid (slice expansion)
			if argSlice, ok := argTypes[i].(*types.Slice); ok {
				if types.Identical(argSlice.Elem(), paramType) {
					// Slice element type matches variadic element type - allow expansion
					continue
				}
			}
		} else {
			paramType = funcType.Params().At(i).Type()
		}

		argumentType = argTypes[i] //nolint:gosec // bounds checked by loop condition

		if argFuncType, ok := argumentType.(*types.Signature); ok {
			retVals, _, localErrs := makeFunctionTypeCheck(
				argFuncType,
				argSymbols[i],
				[]types.Type{},
				[]*lexer.Token{},
				makeTypeInference,
			) //nolint:gosec // bounds checked

			if localErrs != nil {
				errs = append(errs, localErrs...)
				continue
			}
			argumentType = unTuple(retVals)[0]
		}

		// type inference processing for argument of type 'any'
		// BUG: WIP
		reconfigTriggeredByBuiltin := func(paramType types.Type) types.Type {
			tParam, ok := paramType.(*types.TypeParam)
			_ = tParam
			if !ok {
				return paramType
			}
			return nil
		}

		if types.Identical(argumentType, typeAny.Type()) {
			symbol := argSymbols[i] //nolint:gosec // bounds checked

			isBuiltinFunc := false
			if isBuiltinFunc {
				typ := reconfigTriggeredByBuiltin(paramType)

				fromArrayToSlice := func(typ types.Type) types.Type {
					return typ
				}

				paramType = typ
				argumentType = fromArrayToSlice(argumentType)
			}

			recheck, err := makeTypeInference(symbol, argumentType, paramType)
			if err != nil {
				errs = append(errs, err)
			}

			if recheck != nil {
				variablesToRecheck = append(variablesToRecheck, recheck)
			}

			continue
		}

		_, errMsg := TypeCheckAgainstConstraint(argumentType, paramType)
		if errMsg != nil {
			err := parser.NewParseError(
				argSymbols[i],
				errMsg,
			) //nolint:gosec // bounds checked
			errs = append(errs, err)
		}
	}

	// Special handling for comparison functions: propagate types from literals to variables
	if isComparisonFunction(string(funcSymbol.Value)) && len(argSymbols) >= 2 {
		rechecks := inferTypesFromComparison(argSymbols, argTypes, makeTypeInference)
		variablesToRecheck = append(variablesToRecheck, rechecks...)
	}

	// 2. Check Validity for Return Type
	returnSize := funcType.Results().Len()
	if returnSize > 2 {
		err := parser.NewParseError(funcSymbol, errFunctionMaxReturn)
		errs = append(errs, err)
	} else if returnSize == 2 {
		secondReturnType := funcType.Results().At(1).Type()
		errorType := typeError.Type()

		if !types.Identical(secondReturnType, errorType) {
			err := parser.NewParseError(funcSymbol, errFunctionSecondReturnNotError)
			errs = append(errs, err)
		}
	} else if returnSize == 0 {
		err := parser.NewParseError(funcSymbol, errFunctionVoidReturn)
		errs = append(errs, err)
	}

	return funcType.Results(), variablesToRecheck, errs
}

func getVariableDefinitionForRootField(
	variable *lexer.Token,
	localVariables, globalVariables map[string]*VariableDefinition,
) (*VariableDefinition, *parser.ParseError) {
	fields, _, _, err := splitVariableNameFields(variable)
	if err != nil {
		return nil, err
	}

	// 2. Find whether the root variable exists or not
	variableName := fields[0]

	defLocal, foundLocal := localVariables[variableName]
	defGlobal, foundGlobal := globalVariables[variableName]

	var varDef *VariableDefinition

	if foundLocal {
		varDef = defLocal
	} else if foundGlobal {
		varDef = defGlobal
	} else {
		err := parser.NewParseError(variable, errVariableUndefined)
		err.Range.End.Character = err.Range.Start.Character + len(fields[0])
		return nil, err
	}

	return varDef, nil
}

// isComparisonFunction returns true if the function name is a comparison function.
func isComparisonFunction(name string) bool {
	switch name {
	case "eq", "ne", "lt", "le", "gt", "ge":
		return true
	}
	return false
}

// inferTypesFromComparison propagates types from literals to any-typed variables
// in comparison function calls. For example, in `eq .Status "active"`, if .Status
// is type `any`, it will infer that .Status is type `string` from the literal.
func inferTypesFromComparison(
	argSymbols []*lexer.Token,
	argTypes []types.Type,
	makeTypeInference InferenceFunc,
) []*collectionPostCheckImplicitTypeNode {
	var rechecks []*collectionPostCheckImplicitTypeNode

	// Find pairs where one is any-typed and another is a concrete type
	for i := range argSymbols {
		if !types.Identical(argTypes[i], typeAny.Type()) {
			continue // Skip if not any-typed
		}

		// This argument is any-typed; look for a concrete type in other args
		for j := range argSymbols {
			if i == j {
				continue
			}

			// Check if argTypes[j] is a concrete type (not any, not invalid)
			if argTypes[j] == nil || types.Identical(argTypes[j], typeAny.Type()) {
				continue
			}
			if types.Identical(argTypes[j], types.Typ[types.Invalid]) {
				continue
			}

			// Found a concrete type - propagate it to the any-typed argument
			recheck, _ := makeTypeInference(argSymbols[i], argTypes[i], argTypes[j])
			if recheck != nil {
				rechecks = append(rechecks, recheck)
			}
			break // Only need to infer once per any-typed argument
		}
	}

	return rechecks
}
