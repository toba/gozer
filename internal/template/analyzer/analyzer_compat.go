package analyzer

import (
	"fmt"
	"go/types"
	"log"

	"github.com/pacer/gozer/internal/template/lexer"
)

// TypeCheckAgainstConstraint checks if candidateType is compatible with constraintType.
func TypeCheckAgainstConstraint(
	candidateType, constraintType types.Type,
) (types.Type, error) {
	if types.Identical(constraintType, typeAny.Type()) {
		return candidateType, nil
	}

	switch receiver := constraintType.(type) {
	case *types.TypeParam:
		it, ok := receiver.Constraint().Underlying().(*types.Interface)
		if !ok {
			panic(
				"type checker expected an 'interface' within 'type parameter'. type = " + constraintType.String(),
			)
		}

		if types.Satisfies(candidateType, it) {
			return candidateType, nil
		}

	default:
		if types.AssignableTo(candidateType, constraintType) {
			return candidateType, nil
		}
	}

	err := fmt.Errorf(
		"%w, expected '%s' but got '%s'",
		errTypeMismatch,
		constraintType,
		candidateType,
	)
	return types.Typ[types.Invalid], err
}

// TypeCheckCompatibilityWithConstraint checks if expressionType has all fields/methods of templateType.
func TypeCheckCompatibilityWithConstraint(
	expressionType, templateType types.Type,
) (types.Type, error) {
	// When expression type is any (unknown), silently continue rather than erroring.
	// This avoids false positives in templates without explicit type hints.
	if types.Identical(expressionType, typeAny.Type()) {
		return typeAny.Type(), nil
	} else if types.Identical(templateType, typeAny.Type()) {
		return typeAny.Type(), nil
	}

	if types.Identical(expressionType, templateType) {
		return expressionType, nil
	}

	// way to go
	// 1. transform expressionType & templateType to implicitType tree
	// 2. compare the path of those two tree only whenever children are found
	// 3. If we have reached the leave of the tree, compare both the path and the type

	constraintTree := convertTypeToImplicitType(templateType)
	exprTree := convertTypeToImplicitType(expressionType)

	typ, err := checkImplicitTypeCompatibility(exprTree, constraintTree, "$")
	return typ, err
}

func convertTypeToImplicitType(sourceType types.Type) *nodeImplicitType {
	if sourceType == nil {
		log.Printf("unable to convert <nil> type to an implicit type tree")
		panic("unable to convert <nil> type to an implicit type tree")
	}

	parentNode := newNodeImplicitType("<PARENT_NODE>", sourceType, lexer.Range{})

	switch typ := sourceType.(type) {
	case *types.Named: // tree's leave
		for method := range typ.Methods() {
			fieldName := method.Name()
			fieldType := method.Signature()

			node := newNodeImplicitType(fieldName, fieldType, lexer.Range{})
			parentNode.children[node.fieldName] = node
		}
	case *types.Struct:
		for currentField := range typ.Fields() {
			node := convertTypeToImplicitType(currentField.Type())
			node.fieldName = currentField.Name()

			parentNode.children[node.fieldName] = node
		}

	case *types.Alias:
		parentNode = convertTypeToImplicitType(types.Unalias(typ))
	default: // tree's leave
		parentNode.fieldType = typ
		parentNode.fieldName = "<LEAVE_NODE>"
	}

	return parentNode
}

func checkImplicitTypeCompatibility(
	candidateTree, constraintTree *nodeImplicitType,
	rootPath string,
) (types.Type, error) {
	if candidateTree == nil {
		panic("<nil> value for 'candidateTree' within 'checkImplicitTypeCompatibility()'")
	} else if constraintTree == nil {
		panic(
			"<nil> value for 'constraintTree' within 'checkImplicitTypeCompatibility()'",
		)
	}

	// 1. Whenever reaching tree's leave, check that the type match
	if len(constraintTree.children) == 0 {
		typ, err := TypeCheckAgainstConstraint(
			candidateTree.fieldType,
			constraintTree.fieldType,
		)
		if err != nil {
			return typ, fmt.Errorf("%w for field '%s'", err, rootPath)
		}

		return typ, nil
	}

	// 2. Check that every field in constraintTree are also present into candidateTree
	for childName, childNode := range constraintTree.children {
		_, ok := candidateTree.children[childName]

		if !ok {
			return types.Typ[types.Invalid], fmt.Errorf(
				"%w, field not found: '%s' of type '%s'",
				errTypeMismatch,
				rootPath+"."+childName,
				childNode.fieldType,
			)
		}
	}

	// 3. Now go check one level deeper
	for childName, childNode := range constraintTree.children {
		newRootPath := rootPath + "." + childName
		candidateChildNode := candidateTree.children[childName]

		typ, err := checkImplicitTypeCompatibility(
			candidateChildNode,
			childNode,
			newRootPath,
		)
		if err != nil {
			return typ, err
		}
	}

	return constraintTree.fieldType, nil
}

// resolve/obtain, resolveKeyTypeFromInterableType()
// Also, 'iterator' type is still not working properly ! to improve in the future
func getKeyAndValueTypeFromIterableType(
	source types.Type,
) (key, value types.Type, err error) {
	source = source.Underlying()

	if types.Identical(source, typeAny.Type()) {
		return typeAny.Type(), typeAny.Type(), nil
	}

	switch typ := source.(type) {
	case *types.Slice:
		return types.Typ[types.Int], typ.Elem(), nil

	case *types.Map:
		return typ.Key(), typ.Elem(), nil

	case *types.Array:
		return types.Typ[types.Int], typ.Elem(), nil

	case *types.Basic:
		if !types.Identical(typ, types.Typ[types.Int]) {
			break
		}

		return types.Typ[types.Invalid], types.Typ[types.Int], nil

	case *types.Chan:
		return types.Typ[types.Int], typ.Elem(), nil

	// case *types.Named:
	//
	// look for iter.seq & iter.seq2 method
	case *types.Signature: // handle iter.seq & iter.seq2 here
		// 1. verify that return type match iter.seq definition
		if typ.Results().Len() != 1 {
			break // will return an error
		}

		returnType := typ.Results().At(0).Type().Underlying()
		if !types.Identical(returnType, types.Typ[types.Bool]) {
			break // will return an error
		}

		// 2. verify that parameter size match iter.seq definition
		// TODO: continue handling iterator
		// however, I should fetch the definition of the iterator within the std
		// then compare the std version against the user version to see if there is match
		// eg. types.Identical(stdIteratorSignature, typ)
		// For now return any_type

		return typeAny.Type(), typeAny.Type(), nil

	default:
		break // will return an error
	}

	err = fmt.Errorf(
		"%w, expected array, slice, map, int, chan, or iterator",
		errTypeMismatch,
	)

	return types.Typ[types.Invalid], types.Typ[types.Invalid], err
}

// End
