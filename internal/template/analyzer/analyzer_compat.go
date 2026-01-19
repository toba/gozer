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

	// Handle iter.Seq[V] and iter.Seq2[K,V] iterator types
	// iter.Seq[V] = func(yield func(V) bool)
	// iter.Seq2[K,V] = func(yield func(K, V) bool)
	case *types.Signature:
		// 1. Verify outer function has no return value (iter.Seq returns nothing)
		if typ.Results().Len() != 0 {
			break // not an iterator
		}

		// 2. Verify outer function has exactly 1 parameter (the yield func)
		if typ.Params().Len() != 1 {
			break // not an iterator
		}

		// 3. Get the yield function signature
		yieldParam := typ.Params().At(0).Type()
		yieldSig, ok := yieldParam.Underlying().(*types.Signature)
		if !ok {
			break // param is not a function
		}

		// 4. Verify yield function returns bool
		if yieldSig.Results().Len() != 1 {
			break // yield must return exactly one value
		}
		if !types.Identical(yieldSig.Results().At(0).Type(), types.Typ[types.Bool]) {
			break // yield must return bool
		}

		// 5. Extract types from yield parameters
		yieldParams := yieldSig.Params()
		switch yieldParams.Len() {
		case 1:
			// iter.Seq[V]: yield(V) - return (any, V)
			// Key type is irrelevant for single-value iterators
			return typeAny.Type(), yieldParams.At(0).Type(), nil
		case 2:
			// iter.Seq2[K,V]: yield(K, V) - return (K, V)
			return yieldParams.At(0).Type(), yieldParams.At(1).Type(), nil
		}
		// Yield with 0 or 3+ params is not a valid iterator

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
