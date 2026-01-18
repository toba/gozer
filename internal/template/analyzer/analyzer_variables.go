package analyzer

import (
	"bytes"
	"errors"
	"fmt"
	"go/types"
	"log"
	"strconv"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

func definitionAnalysisVariableDeclaration(
	node *parser.VariableDeclarationNode,
	parentScope *parser.GroupStatementNode,
	file *FileDefinition,
	globalVariables, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	if parentScope == nil {
		panic(
			"'variable declaration' cannot be parentless; it should be contained in at least one scope",
		)
	} else if node.Kind() != parser.KindVariableDeclaration {
		log.Printf(
			"found value mismatch for 'VariableDeclarationNode.Kind' during DefinitionAnalysis()"+"\n node = %#v\n parent = %#v\n",
			node,
			parentScope,
		)
		panic(
			"found value mismatch for 'VariableDeclarationNode.Kind' during DefinitionAnalysis()",
		)
	} else if localVariables == nil || globalVariables == nil {
		log.Printf(
			"either 'localVariables' or 'globalVariables' shouldn't be nil for 'VariableDeclarationNode.DefinitionAnalysis()'"+"\n localVariables = %#v\n globalVariables = %#v\n",
			localVariables,
			globalVariables,
		)
		panic(
			"either 'localVariables' or 'globalVariables' shouldn't be nil for 'VariableDeclarationNode.DefinitionAnalysis()'",
		)
	}

	var errs []lexer.Error
	var expressionType [2]types.Type
	var localInferences InferenceFoundReturn
	invalidTypes := [2]types.Type{
		types.Typ[types.Invalid],
		typeError.Type(),
	}

	if node.Err != nil {
		return invalidTypes, localInferences, nil
	}

	if len(node.VariableNames) == 0 && len(node.VariableNames) > 2 {
		log.Printf(
			"cannot analyze variable declaration with 0 or more than 2 variables; this error must be caught and discarded while parsing"+"\n node = %#v\n",
			node,
		)
		panic(
			"cannot analyze variable declaration with 0 or more than 2 variables; this error must be caught and discarded while parsing",
		)
	}

	// 0. Check that variable names have proper syntax
	for _, variable := range node.VariableNames {
		if bytes.ContainsAny(variable.Value, ".") {
			err := parser.NewParseError(
				variable,
				errors.New("forbidden '.' in variable name while declaring"),
			)
			errs = append(errs, err)

			return invalidTypes, localInferences, errs
		}
	}

	// 1. Check that 'expression' is valid
	if node.Value != nil {
		var localErrs []lexer.Error

		expressionType, localInferences, localErrs = definitionAnalysisMultiExpression(
			node.Value,
			parentScope,
			file,
			globalVariables,
			localVariables,
		)
		errs = append(errs, localErrs...)
	} else {
		localErr := parser.NewParseError(
			&lexer.Token{},
			errors.New("assignment expression cannot be empty"),
		)
		localErr.Range = node.Range()
		errs = append(errs, localErr)

		return invalidTypes, localInferences, errs
	}

	// All the code blow suppose that:   len(node.VariableNames) > 0 && len(node.VariableNames) <= 2
	if parentScope.Kind() == parser.KindRangeLoop && parentScope.IsProcessingHeader {
		// helper function
		createRecheckNode := func(def *VariableDefinition, variable *lexer.Token) *collectionPostCheckImplicitTypeNode {
			varName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
			fakeTree := newNodeImplicitType(varName, typeAny.Type(), variable.Range)
			return newCollectionPostCheckImplicitTypeNode(
				def.TreeImplicitType,
				fakeTree,
				def,
				nil,
				variable,
				nil,
			)
		}

		computedExpressionType := expressionType[0]

		if types.Identical(computedExpressionType, typeAny.Type()) &&
			localInferences.uniqueVariableInExpression != nil &&
			types.Identical(
				localInferences.uniqueVariableInExpression.candidateDef.typ,
				typeAny.Type(),
			) {
			rhs := localInferences.uniqueVariableInExpression
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				rhs.candidateDef,
				rhs.candidateSymbol,
			)

			// this code below is mandatory for 'insertIterableIntoImplicitTypeNode()' used later to work properly
			computedExpressionType = exprTree.fieldType // very important

			// NOTE:  in case 'exprTree.toDiscard == true', we do not to handle it here
			// this error is handled in 'makeSymboleDefinitionAnalysis()' for '$' variable
		}

		key, val, errMsg := getKeyAndValueTypeFromIterableType(
			computedExpressionType,
		) // important
		if errMsg != nil {
			err := parser.NewParseError(&lexer.Token{}, errMsg)
			err.Range = node.Value.Range()
			errs = append(errs, err)
		}

		if types.Identical(key, types.Typ[types.Invalid]) &&
			len(node.VariableNames) == 2 {
			err := parser.NewParseError(
				node.VariableNames[0],
				fmt.Errorf("'%s' type does not accept key", computedExpressionType),
			)
			errs = append(errs, err)
		}

		var keyDefinition, valueDefinition *VariableDefinition
		var keyToken, valueToken *lexer.Token
		firstPass := true

		for index := len(node.VariableNames) - 1; index >= 0; index-- {
			variable := node.VariableNames[index]
			variableName := string(variable.Value)

			if _, found := localVariables[variableName]; found {
				err := parser.NewParseError(variable, errVariableRedeclaration)
				errs = append(errs, err)
				continue
			}

			def := NewVariableDefinition(variableName, node, parentScope, file.Name())
			def.rng.Start = variable.Range.Start

			localVariables[variableName] = def

			if firstPass {
				def.typ = val
				valueDefinition = def
				valueToken = variable
				firstPass = false
				continue
			}

			def.typ = key
			keyDefinition = def
			keyToken = variable
			file.secondaryVariable = def
		}

		// type inference of the loop expression variable to be completed latter (arr, slice, map, ...)
		rhs := localInferences.uniqueVariableInExpression
		if types.Identical(computedExpressionType, typeAny.Type()) && rhs != nil &&
			types.Identical(
				rhs.candidateDef.typ,
				typeAny.Type(),
			) { // this condition is to make sure that the tree is still modifiable (type any)
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				rhs.candidateDef,
				rhs.candidateSymbol,
			)
			err := insertIterableIntoImplicitTypeNode(
				exprTree,
				keyDefinition,
				valueDefinition,
			)
			if err != nil {
				err.Token = rhs.candidateSymbol
				err.Range = rhs.candidateSymbol.Range
				errs = append(errs, err)
			}
		}
		// Note: When computedExpressionType is any and we can't do type inference,
		// we silently continue with any type rather than reporting an error.
		// This avoids false positives in templates without explicit type hints.

		// solve issue when 'def.TreeImplicitType == nil'
		_ = extractOrInsertTemporaryImplicitTypeFromVariable(valueDefinition, valueToken)

		recheck := createRecheckNode(valueDefinition, valueToken)
		localInferences.uniqueVariableInExpression = recheck // used by parent scope to assign to '.' var
		localInferences.variablesToRecheckAtEndOfScope = append(
			localInferences.variablesToRecheckAtEndOfScope,
			recheck,
		) // enable computation of real type at end of scope

		if keyDefinition != nil { // sizeCreatedVariable == 2
			_ = extractOrInsertTemporaryImplicitTypeFromVariable(keyDefinition, keyToken)
			recheck := createRecheckNode(keyDefinition, node.VariableNames[0])
			localInferences.variablesToRecheckAtEndOfScope = append(
				localInferences.variablesToRecheckAtEndOfScope,
				recheck,
			) // enable computation of real type at end of scope
		}

		return expressionType, localInferences, errs
	}

	// else, simple variable declaration
	//
	if len(node.VariableNames) > 1 {
		localErr := parser.NewParseError(
			node.VariableNames[1],
			errors.New("only 'range' loop can declare 2 variables at once"),
		)
		errs = append(errs, localErr)
		return invalidTypes, localInferences, errs
	}

	// simple var declaration check (only 1 variable at a time)
	variable := node.VariableNames[0]
	variableName := string(variable.Value)

	if _, found := localVariables[variableName]; found {
		err := parser.NewParseError(variable, errVariableRedeclaration)
		errs = append(errs, err)
		return invalidTypes, localInferences, errs
	}

	def := NewVariableDefinition(variableName, node, parentScope, file.Name())
	def.rng.Start = variable.Range.Start
	def.typ = expressionType[0]

	localVariables[variableName] = def

	// Handle the case when an expression come with an 'inferred type tree'
	// So that at inference type resolution, both variable share the same type
	var recheck *collectionPostCheckImplicitTypeNode = nil

	if types.Identical(def.typ, typeAny.Type()) {
		rhs := localInferences.uniqueVariableInExpression

		var candidate, constraint *nodeImplicitType
		var candidateSymbol *lexer.Token
		var candidateDef *VariableDefinition

		if rhs == nil ||
			(rhs != nil && !types.Identical(rhs.candidateDef.typ, typeAny.Type())) {
			// this help enforcing that the variable remain 'any_type' at end of scope, otherwise an error will show up

			varName := "$_fake_tree_enforce_any_type"
			exprSymbol := lexer.NewToken(
				lexer.DollarVariable,
				node.Value.Range(),
				[]byte(varName),
			)

			fakeExprDef := NewVariableDefinition(
				varName,
				node.Value,
				parentScope,
				file.name,
			)
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				fakeExprDef,
				exprSymbol,
			) // this is a candidate node
			varTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				def,
				variable,
			) // this is the constraint, only for this special case

			candidate = exprTree
			constraint = varTree
			candidateSymbol = exprSymbol
			candidateDef = fakeExprDef
		} else if rhs != nil && types.Identical(rhs.candidateDef.typ, typeAny.Type()) {
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				rhs.candidateDef,
				rhs.candidateSymbol,
			) // this is the candidate node
			def.TreeImplicitType = exprTree

			candidate = exprTree
			constraint = exprTree
			candidateSymbol = variable
			candidateDef = def
		}

		recheck = newCollectionPostCheckImplicitTypeNode(
			candidate,
			constraint,
			candidateDef,
			nil,
			candidateSymbol,
			nil,
		)
		localInferences.variablesToRecheckAtEndOfScope = append(
			localInferences.variablesToRecheckAtEndOfScope,
			recheck,
		)
	}

	if recheck == nil {
		varName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
		fakeTree := newNodeImplicitType(varName, typeAny.Type(), variable.Range)
		varTree := extractOrInsertTemporaryImplicitTypeFromVariable(
			def,
			variable,
		) // this only work bc of assignment rule on 'variable' token
		recheck = newCollectionPostCheckImplicitTypeNode(
			varTree,
			fakeTree,
			def,
			nil,
			variable,
			nil,
		)
	}

	localInferences.uniqueVariableInExpression = recheck

	return expressionType, localInferences, errs
}

func definitionAnalysisVariableAssignment(
	node *parser.VariableAssignationNode,
	parent *parser.GroupStatementNode,
	file *FileDefinition,
	globalVariables, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	if parent == nil {
		panic(
			"'variable declaration' cannot be parentless; it should be contained in at least one scope",
		)
	} else if node.Kind() != parser.KindVariableAssignment {
		panic(
			"found value mismatch for 'VariableAssignationNode.Kind' during DefinitionAnalysis()\n" + node.String(),
		)
	} else if globalVariables == nil || localVariables == nil {
		panic(
			"'localVariables' or 'globalVariables' shouldn't be empty for 'VariableAssignationNode.DefinitionAnalysis()'",
		)
	}

	var errs []lexer.Error
	var assignmentType, expressionType [2]types.Type
	var localInferences InferenceFoundReturn
	invalidTypes := [2]types.Type{
		types.Typ[types.Invalid],
		typeError.Type(),
	}

	if node.Err != nil {
		return invalidTypes, localInferences, nil
	}

	if len(node.VariableNames) == 0 && len(node.VariableNames) > 2 {
		panic(
			"cannot analyze variable declaration with 0 or more than 2 variables; this error must be caught and discarded while parsing. node = " + node.String(),
		)
	}

	// 0. Check that variable names have proper syntax
	for _, variable := range node.VariableNames {
		if bytes.ContainsAny(variable.Value, ".") {
			err := parser.NewParseError(
				variable,
				errors.New("forbidden '.' in variable name while declaring"),
			)
			errs = append(errs, err)
			return invalidTypes, localInferences, errs
		}
	}

	// 1. Check that 'expression' is valid
	if node.Value != nil {
		var localErrs []lexer.Error
		expressionType, localInferences, localErrs = definitionAnalysisMultiExpression(
			node.Value,
			parent,
			file,
			globalVariables,
			localVariables,
		)
		errs = append(errs, localErrs...)
	} else {
		errLocal := parser.NewParseError(
			&lexer.Token{},
			errors.New("assignment value cannot be empty"),
		)
		errLocal.Range = node.Range()
		errs = append(errs, errLocal)
		return invalidTypes, localInferences, errs
	}

	// 2. variable within for loop
	if parent.Kind() == parser.KindRangeLoop && parent.IsProcessingHeader {
		computedExpressionType := expressionType[0]

		rhs := localInferences.uniqueVariableInExpression
		if types.Identical(computedExpressionType, typeAny.Type()) && rhs != nil &&
			types.Identical(rhs.candidateDef.typ, typeAny.Type()) {
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				rhs.candidateDef,
				rhs.candidateSymbol,
			)
			computedExpressionType = exprTree.fieldType
		}

		key, _, errMsg := getKeyAndValueTypeFromIterableType(
			computedExpressionType,
		) // important
		if errMsg != nil {
			err := parser.NewParseError(&lexer.Token{}, errMsg)
			err.Range = node.Value.Range()
			errs = append(errs, err)
		}

		if types.Identical(key, types.Typ[types.Invalid]) &&
			len(node.VariableNames) == 2 {
			err := parser.NewParseError(
				node.VariableNames[0],
				fmt.Errorf("'%s' type does not accept key", computedExpressionType),
			)
			errs = append(errs, err)
		}

		var keyDefinition, valueDefinition *VariableDefinition
		var keyToken, valueToken *lexer.Token
		firstPass := true

		for index := len(node.VariableNames) - 1; index >= 0; index-- {
			variable := node.VariableNames[index]

			def, err := getVariableDefinitionForRootField(
				variable,
				localVariables,
				globalVariables,
			)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if firstPass {
				firstPass = false
				valueToken = variable

				// the only reason why I cloned 'def' was to provide a better 'go-to-definition' within the 'range' loop only
				// a) share the same 'def.TreeImplicitType' pointer, so the update in one will affect the other
				// b) since 'def.typ' is known at declaration, it will not change till the end analysis. So assuming both definition have the same type is safe
				valueDefinition = cloneVariableDefinition(def)
				valueDefinition.rng = variable.Range
				continue
			}

			keyToken = variable
			keyDefinition = def
			file.secondaryVariable = def
		}

		// rhs := localInferences.uniqueVariableInExpression
		if rhs == nil {
			if types.Identical(computedExpressionType, typeAny.Type()) && rhs == nil {
				err := parser.NewParseError(&lexer.Token{}, errDefeatedTypeSystem)
				err.Range = node.Value.Range()
				errs = append(errs, err)
			}

			varFakeName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
			exprSymbol := lexer.NewToken(
				lexer.DollarVariable,
				node.Range(),
				[]byte("$_fake_expr_variable"),
			)
			exprDef := NewVariableDefinition(
				varFakeName,
				node.Value,
				parent,
				file.FileName(),
			)
			exprDef.typ = computedExpressionType
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				exprDef,
				exprSymbol,
			)
			rhs = newCollectionPostCheckImplicitTypeNode(
				exprTree,
				exprTree,
				exprDef,
				nil,
				exprSymbol,
				nil,
			)
		}

		// setup phase for 'keyDefinition' and 'valueDefinition' evaluation
		exprTree := rhs.candidate
		exprDef := rhs.candidateDef
		exprSymbol := rhs.candidateSymbol

		// Post recheck 'key' var and 'expr' iterable key type, when available
		if keyDefinition != nil {
			keyTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				keyDefinition,
				keyToken,
			)
			recheck := newCollectionPostCheckImplicitTypeNode(
				exprTree,
				keyTree,
				exprDef,
				keyDefinition,
				exprSymbol,
				keyToken,
			)
			recheck.operation = operatorKeyIterableType
			localInferences.variablesToRecheckAtEndOfScope = append(
				localInferences.variablesToRecheckAtEndOfScope,
				recheck,
			)
		}

		// Post recheck 'value' var and 'expr' iterable value type
		if valueDefinition != nil {
			valueTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				valueDefinition,
				valueToken,
			)
			recheck := newCollectionPostCheckImplicitTypeNode(
				exprTree,
				valueTree,
				exprDef,
				valueDefinition,
				exprSymbol,
				valueToken,
			)
			recheck.operation = operatorValueIterableType
			localInferences.variablesToRecheckAtEndOfScope = append(
				localInferences.variablesToRecheckAtEndOfScope,
				recheck,
			)
			localInferences.uniqueVariableInExpression = recheck // used by parent scope to assign to '.' var
		}

		return expressionType, localInferences, errs
	}

	// 3. only one variable, and not within for loop
	if len(node.VariableNames) > 1 {
		localErr := parser.NewParseError(
			node.VariableNames[1],
			errors.New("only 'range' loop can declare 2 variables at once"),
		)
		errs = append(errs, localErr)
		return invalidTypes, localInferences, errs
	}

	variable := node.VariableNames[0]
	def, err := getVariableDefinitionForRootField(
		variable,
		localVariables,
		globalVariables,
	)
	if err != nil {
		errs = append(errs, err)
		return invalidTypes, localInferences, errs
	}

	assignmentType[0] = def.typ
	assignmentType[1] = expressionType[1]

	// Whenever implicit type are found for either 'var' or 'expr'
	// simply, recheck the type later at end of root scope
	rhs := localInferences.uniqueVariableInExpression
	if types.Identical(expressionType[0], typeAny.Type()) && rhs != nil ||
		types.Identical(def.typ, typeAny.Type()) && def.TreeImplicitType != nil {
		varDef := def
		varTree := extractOrInsertTemporaryImplicitTypeFromVariable(varDef, variable)

		if rhs == nil {
			varName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
			exprSymbol := lexer.NewToken(
				lexer.DollarVariable,
				node.Value.Range(),
				[]byte(varName),
			)
			exprDef := NewVariableDefinition(
				string(exprSymbol.Value),
				node,
				parent,
				file.FileName(),
			)
			exprDef.typ = expressionType[0]
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				exprDef,
				exprSymbol,
			)
			rhs = newCollectionPostCheckImplicitTypeNode(
				exprTree,
				exprTree,
				exprDef,
				nil,
				exprSymbol,
				nil,
			)
		}

		exprTree := rhs.candidate
		exprSymbol := rhs.candidateSymbol
		exprDef := rhs.candidateDef

		recheck := newCollectionPostCheckImplicitTypeNode(
			exprTree,
			varTree,
			exprDef,
			varDef,
			exprSymbol,
			variable,
		)
		recheck.isAssignmentNode = true
		localInferences.variablesToRecheckAtEndOfScope = append(
			localInferences.variablesToRecheckAtEndOfScope,
			recheck,
		)
		localInferences.uniqueVariableInExpression = recheck

		return assignmentType, localInferences, errs
	}

	varTree := extractOrInsertTemporaryImplicitTypeFromVariable(
		def,
		variable,
	) // this only work bc of assignment rule
	varName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
	fakeTree := newNodeImplicitType(varName, typeAny.Type(), variable.Range)

	recheck := newCollectionPostCheckImplicitTypeNode(
		varTree,
		fakeTree,
		def,
		nil,
		variable,
		nil,
	)
	localInferences.uniqueVariableInExpression = recheck

	_, localErr := TypeCheckAgainstConstraint(expressionType[0], def.typ)
	if localErr != nil {
		errMsg := fmt.Errorf(
			"%w, between var '%s' and expr '%s'",
			errTypeMismatch,
			def.Type(),
			expressionType[0],
		)
		err := parser.NewParseError(variable, errMsg)
		errs = append(errs, err)
	}

	return assignmentType, localInferences, errs
}
