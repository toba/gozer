package analyzer

import (
	"fmt"
	"go/token"
	"go/types"
	"log"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

func guessVariableTypeFromImplicitType(varDef *VariableDefinition) types.Type {
	if varDef == nil {
		panic("cannot guess/infer the type of a <nil> variable")
	}

	if !types.Identical(varDef.typ, typeAny.Type()) {
		return varDef.typ
	}

	// reach here only if variable of type 'any'
	inferredType := buildTypeFromTreeOfType(varDef.TreeImplicitType)

	if types.Identical(inferredType, types.Typ[types.Invalid]) && varDef.name == "." {
		inferredType = varDef.typ
	}

	return inferredType
}

func createImplicitTypeFromRealType(
	parentNode *nodeImplicitType,
	convertGoAstPositionToProjectRange func(token.Pos) lexer.Range,
) *nodeImplicitType {
	if parentNode == nil {
		log.Printf("cannot build implicit type tree from <nil> parent node")
		panic("cannot build implicit type tree from <nil> parent node")
	}

	switch typ := parentNode.fieldType.(type) {
	case *types.Named:
		for method := range typ.Methods() {
			rng := convertGoAstPositionToProjectRange(method.Pos())

			childNode := newNodeImplicitType(method.Name(), method.Signature(), rng)
			parentNode.children[childNode.fieldName] = childNode
		}

		parentNode.fieldType = parentNode.fieldType.Underlying()

		_ = createImplicitTypeFromRealType(parentNode, convertGoAstPositionToProjectRange)

		parentNode.fieldType = typ
	case *types.Struct:
		for field := range typ.Fields() {
			rng := convertGoAstPositionToProjectRange(field.Pos())

			childNode := newNodeImplicitType(field.Name(), field.Type(), rng)
			parentNode.children[childNode.fieldName] = childNode

			_ = createImplicitTypeFromRealType(
				childNode,
				convertGoAstPositionToProjectRange,
			)
		}
	default:
		// do nothing
	}

	return parentNode
}

type nodeImplicitType struct {
	toDiscard   bool
	isIterable  bool
	fieldName   string
	fieldType   types.Type
	children    map[string]*nodeImplicitType
	garbageNode map[string]*nodeImplicitType // only useful that handle nasty case created when iterable are involved, do not use it for anything else
	rng         lexer.Range
}

func newNodeImplicitType(
	fieldName string,
	fieldType types.Type,
	reach lexer.Range,
) *nodeImplicitType {
	if fieldType == nil {
		fieldType = typeAny.Type()
	}

	node := &nodeImplicitType{
		isIterable:  false,
		toDiscard:   false,
		fieldName:   fieldName,
		fieldType:   fieldType,
		children:    make(map[string]*nodeImplicitType),
		garbageNode: make(map[string]*nodeImplicitType),
		rng:         reach,
	}

	return node
}

func buildTypeFromTreeOfType(tree *nodeImplicitType) types.Type {
	if tree == nil {
		return typeAny.Type()
	}

	if tree.fieldType == nil {
		tree.fieldType = typeAny.Type()
	}

	if tree.toDiscard {
		return types.Typ[types.Invalid]
	}

	if !types.Identical(tree.fieldType, typeAny.Type()) {
		return tree.fieldType
	}

	if tree.isIterable {
		if len(tree.children) > 2 {
			log.Printf(
				"no iterable implicit node can have more than 2 children"+"\n tree = %#v\n",
				tree,
			)
			panic("no iterable implicit node can have more than 2 children")
		}

		var keyType types.Type = types.Typ[types.Int]

		keyTree := tree.children["key"]
		if keyTree != nil {
			keyType = buildTypeFromTreeOfType(keyTree)
		}

		valueTree := tree.children["value"]
		if valueTree == nil {
			log.Printf(
				"inferred iterable cannot exist without a 'value' node in its type definition"+"\n tree = %#v\n",
				tree,
			)
			panic(
				"inferred iterable cannot exist without a 'value' node in its type definition",
			)
		}

		valueType := buildTypeFromTreeOfType(valueTree)

		var treeType types.Type
		if types.Identical(keyType, types.Typ[types.Int]) { // create slice
			treeType = types.NewSlice(valueType)
		} else { // create map
			treeType = types.NewMap(keyType, valueType)
		}

		return treeType
	}

	// tree.isNotIterable && tree.fieldType != ANY
	if len(tree.children) == 0 {
		return tree.fieldType
	}

	varFields := make([]*types.Var, 0, len(tree.children))

	for _, node := range tree.children {
		if node.toDiscard {
			continue
		}

		fieldType := buildTypeFromTreeOfType(node)
		field := types.NewVar(token.NoPos, nil, node.fieldName, fieldType)

		varFields = append(varFields, field)
	}

	finalType := tree.fieldType

	if len(varFields) > 0 {
		finalType = types.NewStruct(varFields, nil)
	}

	return finalType
}

// Do not affect the type of any node. It either create an new node or return an existing one
// if the node didn't previously exist, it is tagged as 'toDiscard'
func extractOrInsertTemporaryImplicitTypeFromVariable(
	varDef *VariableDefinition,
	symbol *lexer.Token,
) *nodeImplicitType {
	if varDef == nil {
		panic("found <nil> variable definition while trying to insert implicit node type")
	}

	if varDef.TreeImplicitType == nil {
		root := newNodeImplicitType(varDef.name, varDef.typ, varDef.rng)
		varDef.TreeImplicitType = root
	}

	tree := varDef.TreeImplicitType
	fields, _, _, _ := splitVariableNameFields(symbol)

	for index := 1; index < len(fields); index++ {
		if tree.isIterable { // impossible to go deeper, error
			symbolName := string(symbol.Value)
			root := varDef.TreeImplicitType

			node := root.garbageNode[symbolName]
			if node == nil {
				node = newNodeImplicitType(
					symbolName,
					types.Typ[types.Invalid],
					symbol.Range,
				) // very important
				varDef.TreeImplicitType.garbageNode[node.fieldName] = node
			}

			return node
		}

		fieldName := fields[index]
		child, exists := tree.children[fieldName]
		if !exists { // if child not found create it, and then continue business as usual
			child = newNodeImplicitType(fieldName, typeAny.Type(), symbol.Range)
			child.toDiscard = true

			varName, _ := joinVariableNameFields(fields[:index+1])
			child.rng.End.Character = child.rng.Start.Character + len(varName)

			tree.children[fieldName] = child
		}

		tree = child
	}

	return tree
}

func insertIterableIntoImplicitTypeNode(
	tree *nodeImplicitType,
	keyDefinition, valueDefinition *VariableDefinition,
) *parser.ParseError {
	if tree == nil {
		log.Printf(
			"expected an implicit node to insert interable type but found <nil>"+"\n keyDef = %#v \n valDef = %#v\n",
			keyDefinition,
			valueDefinition,
		)
		panic("expected an implicit node to insert interable type but found <nil>")
	} else if valueDefinition == nil {
		log.Printf(
			"found <nil> as iterable value type\n treeNode = %#v\n keyDef = %#v\n",
			tree,
			keyDefinition,
		)
		panic("found <nil> as iterable value type")
	} else if !types.Identical(tree.fieldType, typeAny.Type()) {
		log.Printf(
			"loop 'key' and 'value' have been created expecting <any-type> for the expression\n tree = %s\n keyDef = %s\n valDef = %s\n",
			tree,
			keyDefinition,
			valueDefinition,
		)
		panic(
			"loop 'key' and 'value' have been created expecting <any-type> for the expression, but its type is " + tree.fieldType.String(),
		)
	} else if !types.Identical(valueDefinition.typ, typeAny.Type()) {
		panic(
			"expected <any-type> for loop value in order to trigger type inference, but got " + valueDefinition.typ.String(),
		)
	}

	if tree.isIterable { // if 'iterable' already inferred earlier, use it rather than creating a new one
		if len(tree.children) == 0 {
			log.Printf(
				"no iterable implicit node can have 0 childreen"+"\n tree = %#v\n key = %#v\n value = %#v\n",
				tree,
				keyDefinition,
				valueDefinition,
			)
			panic("no iterable implicit node can have 0 childreen")
		} else if len(tree.children) > 2 {
			log.Printf(
				"no iterable implicit node can have more than 2 children"+"\n tree = %#v\n key = %#v\n value = %#v\n",
				tree,
				keyDefinition,
				valueDefinition,
			)
			panic("no iterable implicit node can have more than 2 children")
		}

		key := tree.children["key"]
		if key != nil && keyDefinition != nil {
			keyDefinition.TreeImplicitType = key
		}

		value := tree.children["value"]
		if value == nil {
			log.Printf(
				"inferred iterable cannot exist without a 'value' node in its type definition"+"\n tree = %#v\n key = %#v\n value = %#v\n",
				tree,
				keyDefinition,
				valueDefinition,
			)
			panic(
				"inferred iterable cannot exist without a 'value' node in its type definition",
			)
		}

		valueDefinition.TreeImplicitType = value
	} else { // tree.isNotIterable, then create a new inferred iterable type
		if hasOnlyDiscardableChildren(tree) {
			clear(tree.children) // very important
		}

		if len(tree.children) > 0 {
			err := parser.NewParseError(
				&lexer.Token{},
				fmt.Errorf(
					"%w, expected array, slice, map, int, chan, or iterator",
					errTypeMismatch,
				),
			)
			err.Range = tree.rng
			return err
		}

		// make sure there is at least a root node for inference to kick in
		tok := lexer.NewToken(lexer.DollarVariable, lexer.Range{}, []byte("$fake_root"))
		if valueDefinition.TreeImplicitType == nil {
			valueTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				valueDefinition,
				tok,
			)
			valueTree.toDiscard = false
			valueDefinition.TreeImplicitType = valueTree
		}

		if keyDefinition != nil && keyDefinition.TreeImplicitType == nil {
			keyTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				keyDefinition,
				tok,
			)
			keyTree.toDiscard = false
			keyDefinition.TreeImplicitType = keyTree
		}

		// if ok (empty children), convert node to iterable type
		tree.isIterable = true

		if keyDefinition != nil {
			tree.children["key"] = keyDefinition.TreeImplicitType
		}

		tree.children["value"] = valueDefinition.TreeImplicitType
	}

	return nil
}

func hasOnlyDiscardableChildren(tree *nodeImplicitType) bool {
	if tree == nil {
		panic(
			"<nil> nodeImplicitType found while looking for its children discard status",
		)
	}

	for _, child := range tree.children {
		if !child.toDiscard {
			return false
		}
	}

	return true
}

// This function assume every tokens have been processed correctly
// Thus if the first token of an expression have been omitted because it was not define,
// it is not the responsibility of this function to handle it
// In fact, the parent function should have not called this, and handle the problem otherwise
// Best results is when the definition analysis worked out properly
//
// 1. Never call this function if the first token (function or varialbe) of the expression is invalid (definition analysis failed)
// 2. If tokens in the middle failed instead (the definition analysis), send the token over anyway, but with 'invalid type',
// the rest will be handled by this function; the hope to always have an accurate depiction of the length of arguments for function/method
// and handle properly the type mismatch
// 3. Send over the file data or at least definition for all function that will be used
//
// In conclusion, this function must always be called with all the token present in the expression, otherwise you will get inconsistent result
// If there is token that are not valid, just alter its type to 'invalid type'
// But as stated earlier, if the first token is rotten, dont even bother calling this function
//

// it removes the added header from the 'position' count
