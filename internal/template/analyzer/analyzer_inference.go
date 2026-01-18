package analyzer

import (
	"errors"
	"fmt"
	"go/types"
	"log"
	"reflect"
	"strings"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

func makeTypeInferenceWhenPossible(
	symbol *lexer.Token,
	symbolType, constraintType types.Type,
	localVariables, globalVariables map[string]*VariableDefinition,
) (recheck *collectionPostCheckImplicitTypeNode, err *parser.ParseError) {
	if types.Identical(constraintType, typeAny.Type()) { // no type inference to make here
		return nil, nil
	}

	// This shouldn't be executed, since 'makeTypeInference()' (this function) is only called
	// by 'makeTypeCheckOnSymbolForFunction()' whenever the argument type is == any
	// this 'symbolType' of this function == any
	if !types.Identical(symbolType, typeAny.Type()) { // if type != ANY_TYPE
		// TODO: remove the code below and panic instead ????
		_, errMsg := TypeCheckAgainstConstraint(symbolType, constraintType)
		if errMsg != nil {
			err := parser.NewParseError(symbol, errMsg)

			return nil, err
		}

		return nil, nil
	}

	switch symbol.ID {
	case lexer.DollarVariable, lexer.DotVariable:
		// do nothing, and postpone processing
	default:
		_, errMsg := TypeCheckAgainstConstraint(symbolType, constraintType)
		if errMsg != nil {
			err := parser.NewParseError(symbol, errMsg)

			return nil, err
		}

		return nil, nil
	}

	// If we reach here, that mean one thing
	// symbolType == ANY_TYPE && constraintType != ANY_TYPE
	//
	// In that case, do type inference implicitly, if '.' var but disallow '$' var

	varDef, err := getVariableDefinitionForRootField(
		symbol,
		localVariables,
		globalVariables,
	)
	if err != nil {
		return nil, err
	}

	if symbol.ID == lexer.DollarVariable { // ID = '.' var
		varFakeName := "$_fake_root"
		constraintTypeTree := newNodeImplicitType(
			varFakeName,
			constraintType,
			symbol.Range,
		)
		candidateTypeTree := extractOrInsertTemporaryImplicitTypeFromVariable(
			varDef,
			symbol,
		)

		recheck := newCollectionPostCheckImplicitTypeNode(
			candidateTypeTree,
			constraintTypeTree,
			varDef,
			nil,
			symbol,
			nil,
		)

		return recheck, nil
	}

	// else 'lexer.DotVariable'
	_, err = updateVariableImplicitType(varDef, symbol, constraintType)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

var errEmptyVariableName error = errors.New("empty variable name")
var errVariableUndefined error = errors.New("variable undefined")
var errVariableRedeclaration error = errors.New("variable redeclaration")
var errFieldNotFound error = errors.New("field or method not found")
var errMapDontHaveChildren error = errors.New("map cannot have children")
var errSliceDontHaveChildren error = errors.New("slice cannot have children")
var errArrayDontHaveChildren error = errors.New("array cannot have children")
var errChannelDontHaveChildren error = errors.New("channel cannot have children")

// split the variable using '.' as a separator
// the returned locations are computed from the front (for the first one) and from the back (for the second)
// this is done since some variable path (namely expanded token) require computing from the back to remain accurate
func splitVariableNameFields(
	variable *lexer.Token,
) ([]string, []int, []int, *parser.ParseError) {
	if variable == nil {
		panic("cannot split variable fields from <nil> token")
	}

	var err *parser.ParseError
	var index, characterCount, counter int

	var fields []string
	var fieldsLocalPosition []int
	var fieldLocalPositionsComputedFromBack []int

	variableName := string(variable.Value)
	LengthVariableName := len(variableName)

	if variableName == "" {
		return nil, nil, nil, parser.NewParseError(variable, errEmptyVariableName)
	}

	if variableName[0] == '.' { // Handle the case of a 'DOT_VARIABLE', which start with '.'
		fields = append(fields, ".")
		fieldsLocalPosition = append(fieldsLocalPosition, characterCount)
		fieldLocalPositionsComputedFromBack = append(
			fieldLocalPositionsComputedFromBack,
			LengthVariableName-1-characterCount,
		)

		characterCount++

		variableName = variableName[1:]

		if variableName == "" {
			return fields, fieldsLocalPosition, fieldLocalPositionsComputedFromBack, nil
		}
	}

	for {
		counter++
		if counter > 1000 {
			log.Printf(
				"current variableName = %s \n fields = %q \n err = %#v \n index = %d",
				variableName,
				fields,
				err,
				index,
			)
			panic("infinite loop detected while spliting variableName")
		}

		index = strings.IndexByte(variableName, '.')

		if index < 0 {
			if variableName == "" {
				err = parser.NewParseError(
					variable,
					errors.New("variable cannot end with '.'"),
				)
				err.Range.Start.Character += characterCount
				err.Range.End = err.Range.Start
				err.Range.End.Character += 1

				return fields, fieldsLocalPosition, fieldLocalPositionsComputedFromBack, err
			}

			index = len(variableName)
		} else if index == 0 {
			err = parser.NewParseError(
				variable,
				errors.New("consecutive '.' found in variable name"),
			)

			err.Range.Start.Character += characterCount
			err.Range.End = err.Range.Start
			err.Range.End.Character += 1

			return fields, fieldsLocalPosition, fieldLocalPositionsComputedFromBack, err
		}

		fields = append(fields, variableName[:index])
		fieldsLocalPosition = append(fieldsLocalPosition, characterCount)
		fieldLocalPositionsComputedFromBack = append(
			fieldLocalPositionsComputedFromBack,
			LengthVariableName-1-characterCount,
		)

		characterCount += index + 1

		if index >= len(variableName) {
			break
		}

		variableName = variableName[index+1:]
	}

	if len(fields) == 0 {
		return nil, nil, nil, parser.NewParseError(variable, errEmptyVariableName)
	}

	return fields, fieldsLocalPosition, fieldLocalPositionsComputedFromBack, nil
}

// join string together to obtain a valid variable name (dollar or dot)
// NOTE: if you need to use it to compute the Range,
// I recommend join from the middle to the end (rather than beginning to middle)
// This workaround was introduced because of 'EXPANDABLE_GROUP', the first field name don't always accurately
// represent the true Range of its associated sub-expression
func joinVariableNameFields(fields []string) (string, *parser.ParseError) {
	length := len(fields)
	if length == 0 {
		err := parser.NewParseError(&lexer.Token{}, errEmptyVariableName)
		return "", err
	}

	// Check for malformated 'field' that will compose the variable name
	for i := range length {
		field := fields[i]

		if i == 0 && field == "." {
			continue
		}

		if field == "" {
			err := parser.NewParseError(&lexer.Token{}, errEmptyVariableName)
			return "", err
		} else if field == "." {
			msg := errors.New("only root field can be equal to '.'")
			err := parser.NewParseError(&lexer.Token{}, msg)
			return "", err
		} else if strings.Contains(field, ".") {
			msg := errors.New("field of a variable cannot contain '.'")
			err := parser.NewParseError(&lexer.Token{}, msg)
			return "", err
		}
	}

	suffix := strings.Join(fields[1:], ".")
	if fields[0] == "." {
		return "." + suffix, nil
	}

	if suffix == "" {
		return fields[0], nil
	}

	return fields[0] + "." + suffix, nil
}

// start checking from the back of the slice because starting from the front
// make finding position of element more complicated for 'lexer.ExpandableGroup'
// func findFieldContainingRange(fields []string, pos lexer.Position) int {
func findFieldContainingRange(fieldPosCountedFromBack []int, pos lexer.Position) int {
	if pos.Line != 0 {
		return 0
	}

	if pos.Character < 0 {
		panic(
			"to search field position within token, a positive relative position of the cursor is mandatory",
		)
	}

	size := len(fieldPosCountedFromBack)

	for index := size - 1; index >= 0; index-- {
		fieldLocation := fieldPosCountedFromBack[index]

		if fieldLocation >= pos.Character {
			return index
		}
	}

	return 0
}

// compute the real type of variable, ignore the inference tree of type
// meaning, computation of 'varDef.typ' rather than 'varDef.TreeImplicitType'
// 'getDeclaredTypeAssociatedToVariable()', which is different from 'inferredType'
func getRealTypeAssociatedToVariable(
	variable *lexer.Token,
	varDef *VariableDefinition,
) (types.Type, *parser.ParseError) {
	invalidType := types.Typ[types.Invalid]

	// 1. Extract names/fields from variable (separated by '.')
	fields, fieldsLocalPosition, _, err := splitVariableNameFields(variable)
	if err != nil {
		return invalidType, err
	}

	if varDef == nil {
		err := parser.NewParseError(variable, errVariableUndefined)
		err.Range.End.Character = err.Range.Start.Character + len(fields[0])
		return invalidType, err
	}

	parentType := varDef.typ
	if parentType == nil {
		return typeAny.Type(), nil
	} else if types.Identical(parentType, typeAny.Type()) {
		return typeAny.Type(), nil
	}

	// 2. Now go down the struct to find out the final variable or method
	if len(fields) == 0 {
		err := parser.NewParseError(variable, errEmptyVariableName)
		return invalidType, err
	} else if len(fields) == 1 {
		return parentType, nil
	}

	var count int
	var fieldName string
	var fieldPos int
	var sizevariable = len(string(variable.Value))

	// parentType ==> type for i = 0 (at start)
	for i := 1; i < len(fields); i++ {
		count++
		if count > 100 {
			log.Printf(
				"infinite loop detected while analyzing fields.\n fields = %q",
				fields,
			)
			panic("infinite loop detected while analyzing fields")
		}

		fieldName = fields[i]
		fieldPos = fieldsLocalPosition[i]
		fieldPosCountedFromBack := (sizevariable) - fieldPos

		// Always check the parent type but always return the last field
		// of the variable without a check
		switch t := parentType.(type) {
		default:
			log.Printf("parentType = %#v \n reflec.TypeOf(parentType) = %s\n"+
				" fields.index = %d ::: fields = %q",
				parentType, reflect.TypeOf(parentType), i, fields,
			)
			panic("parentType not recognized")

		case *types.Basic:
			errBasic := fmt.Errorf(
				"%w, '%s' cannot accept field",
				errTypeMismatch,
				t.String(),
			)
			err = parser.NewParseError(variable, errBasic)
			err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
			err.Range.End.Character = err.Range.Start.Character + len(fieldName)
			err.Range.Start.Line = err.Range.End.Line

			return invalidType, err

		case *types.Named:
			// a. Check that fieldName match the method name
			var method *types.Func
			var foundMethod bool

			for method0 := range t.Methods() {
				method = method0

				if method.Name() != fieldName {
					continue
				}

				foundMethod = true
				break
			}

			if foundMethod {
				parentType = method.Signature()

				continue
			}

			// b. Unpack the underlying type and restart the loop at the current index
			parentType = t.Underlying()

			i--
			continue

		case *types.Struct:
			parentType = nil
			hasAnyTypedField := false

			for field := range t.Fields() {
				// Check if this struct has any 'any'-typed fields (indicates partial inference)
				if types.Identical(field.Type(), typeAny.Type()) {
					hasAnyTypedField = true
				}
				if field.Name() != fieldName {
					continue
				}
				parentType = field.Type()
				break
			}

			if parentType == nil {
				// If the struct has any-typed fields, it's a partially inferred type.
				// Allow accessing unknown fields by returning 'any'.
				if hasAnyTypedField {
					return typeAny.Type(), nil
				}
				err = parser.NewParseError(variable, errFieldNotFound)
				err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
				err.Range.End.Character = err.Range.Start.Character + len(fieldName)
				err.Range.Start.Line = err.Range.End.Line

				return invalidType, err
			}

			continue

		case *types.Alias:
			parentType = types.Unalias(t)

			i--
			continue

		case *types.Pointer:
			parentType = t.Elem()

			i--
			continue

		case *types.Interface:
			// Handle empty interface (any/interface{}) - allow any field access
			// This prevents false positives when the type is unknown
			if t.Empty() || types.Identical(t, typeAny.Type()) {
				return typeAny.Type(), nil
			}

			parentType = nil

			for field := range t.Methods() {
				if field.Name() != fieldName {
					continue
				}

				parentType = field.Type()

				break
			}

			if parentType == nil {
				err = parser.NewParseError(variable, errFieldNotFound)
				err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
				err.Range.End.Character = err.Range.Start.Character + len(fieldName)
				err.Range.Start.Line = err.Range.End.Line

				return invalidType, err
			}

			continue

		case *types.Signature:
			if t.Params().Len() != 0 {
				err = parser.NewParseError(
					variable,
					fmt.Errorf(
						"%w, chained function must be parameterless",
						errFunctionParameterSizeMismatch,
					),
				)
				err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
				err.Range.End.Character = err.Range.Start.Character + len(fieldName)
				err.Range.Start.Line = err.Range.End.Line

				return invalidType, err
			}

			functionResult := t.Results()

			if functionResult == nil {
				err = parser.NewParseError(variable, errFunctionVoidReturn)
				err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
				err.Range.End.Character = err.Range.Start.Character + len(fieldName)
				err.Range.Start.Line = err.Range.End.Line

				return invalidType, err
			}

			if functionResult.Len() > 2 {
				err = parser.NewParseError(variable, errFunctionMaxReturn)
				err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
				err.Range.End.Character = err.Range.Start.Character + len(fieldName)
				err.Range.Start.Line = err.Range.End.Line

				return invalidType, err
			}

			if functionResult.Len() == 2 &&
				!types.Identical(functionResult.At(1).Type(), typeError.Type()) {
				err = parser.NewParseError(variable, errFunctionSecondReturnNotError)
				err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
				err.Range.End.Character = err.Range.Start.Character + len(fieldName)
				err.Range.Start.Line = err.Range.End.Line

				return invalidType, err
			}

			i--
			parentType = functionResult.At(0).Type()

			continue

		case *types.Map:
			err = parser.NewParseError(variable, errMapDontHaveChildren)
			err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
			err.Range.End.Character = err.Range.Start.Character + len(fieldName)
			err.Range.Start.Line = err.Range.End.Line

			return invalidType, err

		case *types.Chan:
			err = parser.NewParseError(variable, errChannelDontHaveChildren)
			err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
			err.Range.End.Character = err.Range.Start.Character + len(fieldName)
			err.Range.Start.Line = err.Range.End.Line

			return invalidType, err

		case *types.Array:
			err = parser.NewParseError(variable, errArrayDontHaveChildren)
			err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
			err.Range.End.Character = err.Range.Start.Character + len(fieldName)
			err.Range.Start.Line = err.Range.End.Line

			return invalidType, err

		case *types.Slice:
			err = parser.NewParseError(variable, errSliceDontHaveChildren)
			err.Range.Start.Character = variable.Range.End.Character - fieldPosCountedFromBack
			err.Range.End.Character = err.Range.Start.Character + len(fieldName)
			err.Range.Start.Line = err.Range.End.Line

			return invalidType, err
			// array, slice, pointer, channel, map, nil, typeParams, Named, Union, tuple, signature, func
		}
	}

	if parentType == nil {
		log.Printf(
			"parent type not found (parentType == nil).\n variable = %s",
			string(variable.Value),
		)
		panic("parent type not found")
	}

	return parentType, nil
}

func markVariableAsUsed(varDef *VariableDefinition) {
	if varDef == nil {
		return
	}

	varDef.IsUsedOnce = true
}

func getVariableImplicitRange(
	varDef *VariableDefinition,
	symbol *lexer.Token,
) *lexer.Range {
	if symbol == nil {
		log.Printf(
			"cannot set implicit type of not existing symbol.\n varDef = %#v",
			varDef,
		)
		panic("cannot set implicit type of not existing symbol")
	}

	if varDef == nil {
		return nil
	}

	if varDef.TreeImplicitType == nil {
		return nil
	}

	currentNode := varDef.TreeImplicitType

	fields, _, _, err := splitVariableNameFields(symbol)

	_ = err

	// NOTE: I decide to not validate the fields[0] on purpose
	// For the reason, look at note within 'getVariableImplicitType()' function

	for index := 1; index < len(fields); index++ {
		fieldName := fields[index]

		childNode, ok := currentNode.children[fieldName]
		if !ok {
			return &currentNode.rng
		}

		currentNode = childNode
	}

	return &currentNode.rng
}

// TODO: rename function 'updateVariableInferredType()', 'insertTypeIntoImplicitTypeNode()', ??????
// 'updateVariableFromPathToImplicitType'
func updateVariableImplicitType(
	varDef *VariableDefinition,
	symbol *lexer.Token,
	symbolType types.Type,
) (types.Type, *parser.ParseError) {
	if symbol == nil {
		log.Printf(
			"cannot set implicit type of not existing symbol.\n varDef = %#v",
			varDef,
		)
		panic("cannot set implicit type of not existing symbol")
	}

	if varDef == nil {
		return types.Typ[types.Invalid], nil
	}

	// Implicit type only work whenever 'varDef.typ == typeAny' only, otherwise the type is specific enough
	// in other word, if type is already known at declaration time, it is useful to try to infer it
	if !types.Identical(varDef.typ, typeAny.Type()) {
		return symbolType, nil
	}

	if symbolType == nil {
		symbolType = typeAny.Type()
	}

	// if 'varDef' type is 'any', then build the implicit type

	fields, _, fieldPosCountedFromBack, err := splitVariableNameFields(symbol)
	if err != nil {
		return types.Typ[types.Invalid], err
	}

	if len(fields) == 0 {
		err = parser.NewParseError(symbol, errEmptyVariableName)
		return types.Typ[types.Invalid], err
	}

	if varDef.TreeImplicitType == nil {
		rootName := fields[0]
		rootType := typeAny.Type()
		rootRange := symbol.Range
		rootRange.End.Character = rootRange.Start.Character + len(rootName)

		varDef.TreeImplicitType = newNodeImplicitType(rootName, rootType, rootRange)
	} else if varDef.TreeImplicitType.toDiscard {
		rootRange := symbol.Range
		rootRange.End.Character = rootRange.Start.Character + len(fields[0])

		varDef.TreeImplicitType.rng = rootRange
		// varDef.TreeImplicitType.fieldName = fields[0] // this one cause a bug for 'with' statement
		varDef.TreeImplicitType.toDiscard = false
	}

	var previousNode *nodeImplicitType
	currentNode := varDef.TreeImplicitType

	// Tree traversal & creation with default value for node in the middle
	// only traverse to the last field in varName
	for index := range len(fields) - 1 {
		if currentNode == nil {
			log.Printf("an existing/created 'implicitTypeNode' cannot be also <nil>"+
				"\n fields = %q\n symbolType = %s\n", fields, symbolType)
			panic("an existing/created 'implicitTypeNode' cannot be also <nil>")
		}

		fieldRange := symbol.Range

		// partialVarName, _ := joinVariableNameFields(fields[:index+1])
		// fieldRange.End.Character = fieldRange.Start.Character + len(partialVarName)

		// NEW ERA
		fieldRange.Start.Character = symbol.Range.End.Character - fieldPosCountedFromBack[index] - 1
		fieldRange.End.Character = fieldRange.Start.Character + len(fields[index])
		// END NEW ERA

		// fieldRange.Start.Character = fieldRange.End.Character - len(fieldName)

		if currentNode.isIterable { // We cannot go deeper into the tree if found in middle of path
			errMsg := fmt.Errorf(
				"%w, expected 'struct' or 'func' but found 'iterable'",
				errTypeMismatch,
			)
			err := parser.NewParseError(symbol, errMsg)
			err.Range = fieldRange

			return currentNode.fieldType, err
		}

		fieldName := fields[index+1]
		childNode, ok := currentNode.children[fieldName]
		if ok {
			if childNode.toDiscard {
				childNode.toDiscard = false
				childNode.rng = fieldRange
			}

			if types.Identical(currentNode.fieldType, typeAny.Type()) {
				previousNode = currentNode
				currentNode = childNode

				continue
			}

			// Check that the remaining variable path is available within the the 'declared type'
			// ie, turn off 'type inference', and look at type directly
			//
			// currentNode.fieldType != ANY
			remainingVarName, _ := joinVariableNameFields(fields[index:])

			varTokenToCheck := lexer.NewToken(
				lexer.DollarVariable,
				symbol.Range,
				[]byte(remainingVarName),
			)
			varTokenToCheck.Range.Start.Character = symbol.Range.End.Character - len(
				remainingVarName,
			)

			fakeVarDef := NewVariableDefinition(
				remainingVarName,
				nil,
				nil,
				"fake_var_definition",
			)
			fakeVarDef.typ = currentNode.fieldType

			fieldType, err := getRealTypeAssociatedToVariable(varTokenToCheck, fakeVarDef)
			if err != nil {
				return currentNode.fieldType, err
			}

			if !types.Identical(fieldType, symbolType) {
				err = parser.NewParseError(symbol, errTypeMismatch)
				err.Range = fieldRange

				return currentNode.fieldType, err
			}

			return currentNode.fieldType, nil
		}

		fieldType := typeAny.Type()

		childNode = newNodeImplicitType(fieldName, fieldType, fieldRange)
		currentNode.children[fieldName] = childNode

		previousNode = currentNode
		currentNode = childNode
	}

	// Last field in variable name
	lastFieldName := fields[len(fields)-1]
	constraintType := symbolType

	lastFieldRange := symbol.Range
	lastFieldRange.Start.Character = lastFieldRange.End.Character - len(lastFieldName)

	// NO NO NO NOOOOO, this is not the time to check a specific child
	// New Strategy:
	//
	// 0. When currentNode == nil, create a new node and exit
	// 1. Check that currentNode type != ANY ===> then compare 'symbolType' and 'currentNode.fieldType'
	// 2. Check that currentNode.fieldType == ANY ======> then 2 situations can occur:
	// 3. len(currentNode.children) == 0 ==========> then currentNode.fieldType = symbolType
	// 4. len(currentNode.children) > 0 ===========> then for each 'child' compute child_type
	//      and make sure it is part of 'symbolType'
	//

	if currentNode == nil {
		currentNode = newNodeImplicitType(lastFieldName, symbolType, lastFieldRange)
		previousNode.children[lastFieldName] = currentNode // safe bc rootNode != nil

		return currentNode.fieldType, nil
	}

	currentNode.toDiscard = false

	if types.Identical(
		constraintType,
		typeAny.Type(),
	) { // do not update node type when received constraintType is ANY_TYPE
		return currentNode.fieldType, nil
	}

	if !types.Identical(currentNode.fieldType, typeAny.Type()) {
		if !types.Identical(currentNode.fieldType, constraintType) {
			errMsg := fmt.Errorf(
				"%w, expected '%s' but got '%s'",
				errTypeMismatch,
				constraintType,
				currentNode.fieldType,
			)

			err = parser.NewParseError(symbol, errMsg)
			err.Range = lastFieldRange

			return currentNode.fieldType, err
		}

		return currentNode.fieldType, nil
	}

	// currentNode.fieldType == ANY_TYPE && len(currentNode.children) == 0
	if len(currentNode.children) == 0 {
		currentNode.fieldType = constraintType

		return currentNode.fieldType, nil
	}

	// currentNode.fieldType == ANY_TYPE && len(currentNode.children) > 0
	typ := buildTypeFromTreeOfType(currentNode)

	if types.Identical(
		typ,
		typeAny.Type(),
	) { // this is for case when childreen are node 'toDiscard'
		currentNode.fieldType = constraintType
		return currentNode.fieldType, nil
	} else if !types.Identical(
		typ,
		constraintType,
	) {
		errMsg := fmt.Errorf(
			"%w, expected '%s' but got '%s'",
			errTypeMismatch,
			constraintType,
			typ,
		)
		err := parser.NewParseError(symbol, errMsg)

		return currentNode.fieldType, err
	}

	currentNode.fieldType = constraintType

	return currentNode.fieldType, nil
}

// Will only guess if variable type is 'any', otherwise return the current type of the variable
