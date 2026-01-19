package analyzer

import (
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	goParser "go/parser"
	"go/token"
	"go/types"
	"log"
	"maps"
	"strconv"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

func analyzeGroupStatementHeader(
	group *parser.GroupStatementNode,
	file *FileDefinition,
	scopedGlobalVariables, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	var controlFlowType [2]types.Type
	var inferences InferenceFoundReturn
	var errs []lexer.Error

	group.IsProcessingHeader = true
	defer func() { group.IsProcessingHeader = false }() // just for safety

	switch group.Kind() {
	case parser.KindIf, parser.KindElseIf, parser.KindRangeLoop, parser.KindWith,
		parser.KindElseWith, parser.KindDefineTemplate, parser.KindBlockTemplate:

		if group.Err != nil { // do not analyze the header
			controlFlowType[0] = types.Typ[types.Invalid]
			break
		}

		if group.ControlFlow == nil {
			log.Printf(
				"fatal, 'controlFlow' not found for 'GroupStatementNode'. \n %s \n",
				group,
			)
			panic(
				"this 'GroupStatementNode' expect a non-nil 'controlFlow' based on its type ('Kind') " + group.Kind().
					String(),
			)
		}

		// The only purpose of this is to ease the computation of '.' var type
		// by embedding 'MultiExpressionNode' within a 'VariableDeclarationNode'
		if group.Kind() == parser.KindRangeLoop {
			if mExpression, ok := group.ControlFlow.(*parser.MultiExpressionNode); ok {
				varName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
				variable := lexer.NewToken(
					lexer.DollarVariable,
					group.ControlFlow.Range(),
					[]byte(varName),
				)

				temporaryVarNode := parser.NewVariableDeclarationNode(
					parser.KindVariableDeclaration,
					group.Range().Start,
					group.Range().End,
					nil,
				)
				temporaryVarNode.Value = mExpression
				temporaryVarNode.VariableNames = append(
					temporaryVarNode.VariableNames,
					variable,
				)

				controlFlowType, inferences, errs = definitionAnalysisRecursive(
					temporaryVarNode,
					group,
					file,
					scopedGlobalVariables,
					localVariables,
				)
				break
			}
		}

		controlFlowType, inferences, errs = definitionAnalysisRecursive(
			group.ControlFlow,
			group,
			file,
			scopedGlobalVariables,
			localVariables,
		)
	}

	group.IsProcessingHeader = false

	return controlFlowType, inferences, errs
}

func definitionAnalysisGroupStatement(
	node *parser.GroupStatementNode,
	_ *parser.GroupStatementNode,
	file *FileDefinition,
	globalVariables, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	if globalVariables == nil || localVariables == nil {
		panic(
			"arguments global/local/function/template definition for 'DefinitionAnalysis()' shouldn't be 'nil' for 'GroupStatementNode'",
		)
	}

	if node.IsRoot() && node.Parent() != nil {
		panic("only root node can be flagged as 'root' and with 'parent == nil'")
	}

	// 1. Variables Init
	scopedGlobalVariables := map[string]*VariableDefinition{}

	maps.Copy(scopedGlobalVariables, globalVariables)
	maps.Copy(scopedGlobalVariables, localVariables)

	localVariables = map[string]*VariableDefinition{} // 'localVariables' lost reference to the parent 'map', so no need to worry using it
	file.secondaryVariable = nil

	var inferences InferenceFoundReturn
	var errs, localErrs []lexer.Error
	var controlFlowType, scopeType [2]types.Type

	scopeType[0] = typeAny.Type()
	scopeType[1] = typeError.Type()

	// 2. ControlFlow analysis
	controlFlowType, inferences, errs = analyzeGroupStatementHeader(
		node,
		file,
		scopedGlobalVariables,
		localVariables,
	)

	// 3. Set Variables Scope
	switch {
	case node.IsGroupWithNoVariableReset():
		// No variable reset needed for if/else/elseif/end

	case node.Kind() == parser.KindRangeLoop:
		var primaryVariable *VariableDefinition

		if inferences.uniqueVariableInExpression == nil {
			primaryVariable = NewVariableDefinition(
				".",
				node,
				node.Parent(),
				file.FileName(),
			)
			primaryVariable.typ = types.Typ[types.Invalid]
		} else {
			primaryVariable = inferences.uniqueVariableInExpression.candidateDef
		}

		localVariables["."] = primaryVariable
		markVariableAsUsed(localVariables["."])

		// Preserve $ from parent scope so it can be accessed inside range blocks.
		// In Go templates, $ always refers to the root data passed to Execute().
		if dollarDef := scopedGlobalVariables["$"]; dollarDef != nil {
			localVariables["$"] = dollarDef
		}

		// if 'key' exists for the current loop, enable type inference for it
		// (this is an exceptional case only found in 'range' loop, since everywhere else '$' var dont have type inference capability)
		if file.secondaryVariable != nil {
			def := file.secondaryVariable
			file.secondaryVariable = nil

			if otherDef := file.extraVariableNameWithTypeInferenceBehavior[def.name]; otherDef != nil { // useful to save and later put back the original value, if it exists in the first place
				defer func() {
					file.extraVariableNameWithTypeInferenceBehavior[def.name] = otherDef
				}()
			}

			file.extraVariableNameWithTypeInferenceBehavior[def.name] = def
		}

	case node.Kind() == parser.KindWith || node.Kind() == parser.KindElseWith:
		localVariables["."] = NewVariableDefinition(
			".",
			node.ControlFlow,
			node,
			file.Name(),
		)
		localVariables["."].typ = controlFlowType[0] //nolint:gosec // controlFlowType is [2]types.Type fixed-size array

		// NOTE: This enable type resolution at end of scope for context variable '.'
		rhs := inferences.uniqueVariableInExpression
		//nolint:gosec // controlFlowType is [2]types.Type fixed-size array
		if types.Identical(controlFlowType[0], typeAny.Type()) &&
			types.Identical(rhs.candidateDef.typ, typeAny.Type()) &&
			rhs != nil {
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				rhs.candidateDef,
				rhs.candidateSymbol,
			)
			varDef := localVariables["."]
			varDef.TreeImplicitType = exprTree

			varName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
			fakeTree := newNodeImplicitType(varName, typeAny.Type(), rhs.candidate.rng)
			symbol := lexer.NewToken(lexer.DotVariable, varDef.rng, []byte("."))

			recheck := newCollectionPostCheckImplicitTypeNode(
				varDef.TreeImplicitType,
				fakeTree,
				varDef,
				nil,
				symbol,
				nil,
			)
			inferences.variablesToRecheckAtEndOfScope = append(
				inferences.variablesToRecheckAtEndOfScope,
				recheck,
			) // enable late type resolution for type inference
			inferences.uniqueVariableInExpression = nil

			// NOTE: this is useful to type check the any 'header' type with the 'inner scope' type
			// the only goal is to manage the case rhs.candidate == typeAny
		} else if types.Identical(
			controlFlowType[0],
			typeAny.Type(),
		) &&
			!types.Identical(rhs.candidateDef.typ, typeAny.Type()) &&
			rhs != nil {
			varDef := localVariables["."]
			varSymbol := lexer.NewToken(
				lexer.DollarVariable,
				varDef.rng,
				[]byte(varDef.name),
			)

			exprDef := rhs.candidateDef
			exprSymbol := rhs.candidateSymbol
			exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(
				rhs.candidateDef,
				rhs.candidateSymbol,
			)
			exprTree.toDiscard = false

			if types.Identical(rhs.candidateDef.typ, typeAny.Type()) {
				varDef.TreeImplicitType = exprTree
			} else {
				varDef.TreeImplicitType = newNodeImplicitType(
					varDef.name,
					typeAny.Type(),
					varDef.rng,
				)
			}

			if types.Identical(
				varDef.TreeImplicitType.fieldType,
				types.Typ[types.Invalid],
			) {
				varDef.typ = types.Typ[types.Invalid]
			}

			recheck := newCollectionPostCheckImplicitTypeNode(
				exprTree,
				varDef.TreeImplicitType,
				exprDef,
				varDef,
				exprSymbol,
				varSymbol,
			)
			inferences.variablesToRecheckAtEndOfScope = append(
				inferences.variablesToRecheckAtEndOfScope,
				recheck,
			) // enable late type resolution for type inference
			inferences.uniqueVariableInExpression = nil
		}

		markVariableAsUsed(localVariables["."])

		// Preserve $ from parent scope so it can be accessed inside with blocks.
		// In Go templates, $ always refers to the root data passed to Execute().
		if dollarDef := scopedGlobalVariables["$"]; dollarDef != nil {
			localVariables["$"] = dollarDef
		}

	case node.IsGroupWithDollarAndDotVariableReset():
		scopedGlobalVariables = make(map[string]*VariableDefinition)
		localVariables = make(map[string]*VariableDefinition)

		localVariables["."] = NewVariableDefinition(".", node, node.Parent(), file.name)
		// Create a separate $ definition so type inference on . doesn't affect $.
		// In Go templates, $ refers to the root data and should maintain type=any
		// so that $.Field accesses work without "field not found" errors.
		localVariables["$"] = NewVariableDefinition("$", node, node.Parent(), file.name)

		markVariableAsUsed(localVariables["."])

		commentGoCode := node.ShortCut.CommentGoCode
		if commentGoCode != nil {
			_, _, localErrs := definitionAnalysisComment(
				commentGoCode,
				node,
				file,
				scopedGlobalVariables,
				localVariables,
			)
			errs = append(errs, localErrs...)
		}

	default:
		panic(
			"found unexpected 'Kind' for 'GroupStatementNode' during 'DefinitionAnalysis()'\n node = " + node.String(),
		)
	}

	// 4. Statements analysis
	var statementType [2]types.Type
	var localInferences InferenceFoundReturn

	for _, statement := range node.Statements {
		if statement == nil {
			panic(
				"statement within 'GroupStatementNode' cannot be nil. make to find where this nil value has been introduced and rectify it",
			)
		}

		// skip already analyzed 'goCode' (done above)
		if statement == node.ShortCut.CommentGoCode {
			continue
		}

		// skip template scope analysis when already done during template dependencies analysis
		scope, isScope := statement.(*parser.GroupStatementNode)
		if isScope && scope.IsTemplate() && file.isTemplateGroupAlreadyAnalyzed {
			if scope.Kind() == parser.KindBlockTemplate { // analyze the header 'expression' before skipping
				_, localInferences, localErrs = analyzeGroupStatementHeader(
					scope,
					file,
					scopedGlobalVariables,
					localVariables,
				)
				errs = append(errs, localErrs...)
				inferences.variablesToRecheckAtEndOfScope = append(
					inferences.variablesToRecheckAtEndOfScope,
					localInferences.variablesToRecheckAtEndOfScope...)
			}

			continue
		}

		// Make DefinitionAnalysis for every children
		statementType, localInferences, localErrs = definitionAnalysisRecursive(
			statement,
			node,
			file,
			scopedGlobalVariables,
			localVariables,
		)
		errs = append(errs, localErrs...)
		inferences.variablesToRecheckAtEndOfScope = append(
			inferences.variablesToRecheckAtEndOfScope,
			localInferences.variablesToRecheckAtEndOfScope...)

		if statementType[1] == nil ||
			types.Identical(statementType[1], typeError.Type()) {
			continue
		}

		err := parser.NewParseError(
			&lexer.Token{},
			fmt.Errorf("%w, second return type must be an 'error' type", errTypeMismatch),
		)
		err.Range = statement.Range()
		errs = append(errs, err)
	}

	// Verify that all 'localVariables' have been used at least once
	for _, def := range localVariables {
		otherDef := file.extraVariableNameWithTypeInferenceBehavior[def.name]
		if otherDef == def {
			delete(file.extraVariableNameWithTypeInferenceBehavior, def.name)
		}

		if def.IsUsedOnce {
			continue
		}

		err := parser.NewParseError(&lexer.Token{}, errVariableNotUsed)
		err.Range = def.Node().Range()
		errs = append(errs, err)
	}

	// Implicitly guess the type of var '.' if no type is found (empty or 'any')
	if node.IsGroupWithDollarAndDotVariableReset() {
		typ := guessVariableTypeFromImplicitType(localVariables["."])
		localVariables["."].typ = typ

		file.typeHints[node] = localVariables["."].typ

		//
		// Check Implicit Node for the end of this scope (type inference resolution)
		//
		for _, recheck := range inferences.variablesToRecheckAtEndOfScope {
			if recheck == nil {
				panic("found unexpect <nil> amongs variable to recheck at end of scope")
			}

			// 1. Test that the whole token is valid, and build the root variable type
			constraintDef := recheck.constraintDef
			if constraintDef != nil {
				constraintDef.typ = guessVariableTypeFromImplicitType(constraintDef)

				tmpNode := extractOrInsertTemporaryImplicitTypeFromVariable(
					constraintDef,
					recheck.constraintSymbol,
				)
				if recheck.constraint != tmpNode &&
					!recheck.constraint.toDiscard { // bc 'discardable' node can later be deleted (eg. inserting iterable when only discardable node are found)
					panic(
						"constraint 'nodeImplicitType' does not match the one coming from its variable definition",
					)
				}
			}

			candidateToken := recheck.candidateSymbol
			candidateDef := recheck.candidateDef

			tmpNode := extractOrInsertTemporaryImplicitTypeFromVariable(
				candidateDef,
				candidateToken,
			)
			if recheck.candidate != tmpNode &&
				!recheck.candidate.toDiscard { // bc 'discardable' node can later be deleted (eg. inserting iterable when only discardable node are found)
				panic(
					"'nodeImplicitType' does not match the one coming from its variable definition",
				)
			}

			// this is to make sure that the variable path exist, as defined by the variable type
			candidateDef.typ = guessVariableTypeFromImplicitType(candidateDef)
			_, err := getRealTypeAssociatedToVariable(candidateToken, candidateDef)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			// 2. Test that the implicit type tree is valid
			candidateType := buildTypeFromTreeOfType(recheck.candidate)
			constraintType := buildTypeFromTreeOfType(recheck.constraint)

			switch recheck.operation {
			case operatorStrictType:
				_, errMsg := TypeCheckAgainstConstraint(candidateType, constraintType)
				if errMsg != nil {
					err := parser.NewParseError(recheck.candidateSymbol, errMsg)
					errs = append(errs, err)
				}

			case operatorCompatibleType:
				_, errMsg := TypeCheckCompatibilityWithConstraint(
					candidateType,
					constraintType,
				)
				if errMsg != nil {
					err := parser.NewParseError(recheck.candidateSymbol, errMsg)
					errs = append(errs, err)
				}

			case operatorKeyIterableType:
				exprTree := recheck.candidate
				keyTree := recheck.constraint

				if keyTree == nil {
					panic(
						"key implicit tree is <nil> while rechecking at end of scope (key of a loop)",
					)
				} else if exprTree == nil {
					panic(
						"expression inference tree is <nil> while rechecking at end of scope (expression of the loop)",
					)
				}

				if !exprTree.isIterable { // no need to return an error while checking the key, since if the key is present, so is the value as well (we will check there)
					continue
				}

				keyType := buildTypeFromTreeOfType(keyTree)

				expectedKeyNode := exprTree.children["key"]
				if expectedKeyNode == nil {
					if types.Identical(keyType, types.Typ[types.Int]) { // success
						continue
					}

					errMsg := fmt.Errorf(
						"%w, expect 'int' but found %s",
						errTypeMismatch,
						keyType.String(),
					)
					err := parser.NewParseError(recheck.constraintSymbol, errMsg)
					errs = append(errs, err)
					continue
				}

				expectedKeyType := buildTypeFromTreeOfType(expectedKeyNode)

				_, errMsg := TypeCheckAgainstConstraint(keyType, expectedKeyType)
				if errMsg != nil {
					err := parser.NewParseError(recheck.constraintSymbol, errMsg)
					errs = append(errs, err)
				}

			case operatorValueIterableType:

				exprTree := recheck.candidate
				valueTree := recheck.constraint

				if valueTree == nil {
					panic(
						"value implicit tree is <nil> while rechecking at end of scope (key of a loop)",
					)
				} else if exprTree == nil {
					panic(
						"expression inference tree is <nil> while rechecking at end of scope (expression of the loop)",
					)
				}

				if !exprTree.isIterable {
					errMsg := fmt.Errorf(
						"%w, expected array, slice, map, int, chan, or iterator",
						errTypeMismatch,
					)
					err := parser.NewParseError(recheck.candidateSymbol, errMsg)
					errs = append(errs, err)
					continue
				}

				valueType := buildTypeFromTreeOfType(valueTree)

				expectedValueNode := exprTree.children["value"]
				if expectedValueNode == nil {
					log.Printf(
						"found <nil> value within iterable 'value'.\n exprTree = %#v\n recheck = %#v\n",
						exprTree,
						recheck,
					)
					panic("found <nil> value within iterable 'value'")
				}

				expectedValueType := buildTypeFromTreeOfType(expectedValueNode)

				_, errMsg := TypeCheckAgainstConstraint(valueType, expectedValueType)
				if errMsg != nil {
					err := parser.NewParseError(recheck.constraintSymbol, errMsg)
					errs = append(errs, err)
				}

			default:
				panic(
					"found unknown operator while rechecking variable type at end of scope",
				)
			}

			// Update shadow type for assignment nodes with any-typed constraints
			if recheck.isAssignmentNode &&
				types.Identical(constraintDef.typ, typeAny.Type()) {
				varDef := constraintDef
				expressionType := candidateType

				if varDef.shadowType == nil ||
					types.Identical(varDef.shadowType, typeAny.Type()) {
					varDef.shadowType = expressionType
				} else if !types.Identical(
					varDef.shadowType,
					typeAny.Type(),
				) {
					_, errMsg := TypeCheckAgainstConstraint(
						expressionType,
						varDef.shadowType,
					)
					if errMsg != nil {
						err := parser.NewParseError(
							recheck.candidateSymbol,
							fmt.Errorf("shadow %w", errMsg),
						)
						errs = append(errs, err)
					}
				}
			}
		}

		// reset all inferences once resolved
		inferences.uniqueVariableInExpression = nil
		inferences.variablesToRecheckAtEndOfScope = nil
	}

	if localVariables["."] != nil {
		scopeType[0] = localVariables["."].typ
	}

	// save all local variables for the current scope; and set the type hint
	file.scopeToVariables[node] = localVariables

	return scopeType, inferences, errs
}

func definitionAnalysisTemplatateStatement(
	node *parser.TemplateStatementNode,
	parent *parser.GroupStatementNode,
	file *FileDefinition,
	globalVariables, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	if parent == nil {
		panic(
			fmt.Sprintf(
				"template cannot be parentless; it should be contained in at least one scope. file = %s :: node = %s",
				file.name,
				node.String(),
			),
		)
	}

	var errs, localErrs []lexer.Error
	var expressionType [2]types.Type
	var inferences InferenceFoundReturn

	invalidTypes := [2]types.Type{
		types.Typ[types.Invalid],
		typeError.Type(),
	}

	if node.Err != nil {
		return invalidTypes, inferences, nil
	}

	if node.TemplateName == nil {
		panic(
			"the template name should never be empty for a template expression. make sure the template has been parsed correctly.\n" + node.String(),
		)
	}

	// 1. Expression analysis, if any
	if node.Expression != nil {
		expressionType, inferences, localErrs = definitionAnalysisRecursive(
			node.Expression,
			parent,
			file,
			globalVariables,
			localVariables,
		)
		errs = append(errs, localErrs...)
	}

	// 2. template name analysis
	switch node.Kind() {
	case parser.KindDefineTemplate: // NOTE: the template definition has already been done in a previous phase (dependency analysis), no need to do again
		if parent.Kind() != node.Kind() {
			panic(
				"value mismatch for 'define' kind. 'TemplateStatementNode.Kind' and 'TemplateStatementNode.parent.Kind' must be similar",
			)
		}

		if !parent.Parent().IsRoot() {
			err := parser.NewParseError(
				node.TemplateName,
				errors.New("template cannot be defined in local scope"),
			)
			errs = append(errs, err)
		}

		if node.Expression != nil {
			err := parser.NewParseError(
				node.TemplateName,
				errors.New("'define' does not accept expression"),
			)
			err.Range = node.Expression.Range()
			errs = append(errs, err)
		}

		inferences.uniqueVariableInExpression = nil

	case parser.KindBlockTemplate: // NOTE: the template definition has already been done in a previous phase (dependency analysis), fallthrough to next case
		if parent.Kind() != node.Kind() {
			panic(
				"value mismatch for 'block' kind. 'TemplateStatementNode.Kind' and 'TemplateStatementNode.parent.Kind' must be similar",
			)
		}

		if !file.isTemplateGroupAlreadyAnalyzed {
			if !parent.Parent().IsRoot() {
				err := parser.NewParseError(
					node.TemplateName,
					errors.New("template cannot be defined in local scope"),
				)
				errs = append(errs, err)
			}

			if node.Expression == nil {
				err := parser.NewParseError(
					node.TemplateName,
					errors.New("missing expression"),
				)
				errs = append(errs, err)
				return invalidTypes, inferences, errs
			}

			break
		}

		// file.isTemplateGroupAlreadyAnalyzed == true
		fallthrough

		// only type check expression after 'template dependency analysis'
		// the second phase is enabled though 'GroupStatementNode', while type checking every children

		/*
			rhs := inferences.uniqueVariableInExpression
			if rhs == nil {
				id := strconv.Itoa(parser.GetUniqueNumber())
				varName := nameTempVar + id
				exprSymbol := lexer.NewToken(lexer.DollarVariable, node.Expression.Range(), []byte(varName))
				exprDef := NewVariableDefinition("$_transit_expr_"+id, node.Expression, parent, file.name)
				exprDef.typ = expressionType[0]
				exprTree := extractOrInsertTemporaryImplicitTypeFromVariable(exprDef, exprSymbol)
				rhs = newCollectionPostCheckImplicitTypeNode(exprTree, exprTree, exprDef, nil, exprSymbol, nil)
			}

			fakeTree := newNodeImplicitType("$_fake_temporary_tree_template", types.Typ[types.Invalid], node.TemplateName.Range)
			recheck := newCollectionPostCheckImplicitTypeNode(rhs.candidate, fakeTree, rhs.candidateDef, nil, rhs.candidateSymbol, nil)
			recheck.operation = operatorCompatibleType

			inferences.uniqueVariableInExpression = recheck
		*/

	case parser.KindUseTemplate:
		templateName := string(node.TemplateName.Value)

		templateDef, found := file.templates[templateName]
		if !found {
			err := parser.NewParseError(node.TemplateName, errTemplateUndefined)
			errs = append(errs, err)
			return invalidTypes, inferences, errs
		}

		if templateDef == nil {
			panic(
				fmt.Sprintf(
					"'TemplateDefinition' cannot be nil for an existing template. file = %#v",
					file,
				),
			)
		} else if templateDef.inputType == nil {
			panic(
				fmt.Sprintf(
					"defined template cannot have 'nil' InputType. def = %#v",
					templateDef,
				),
			)
		}

		if expressionType[0] == nil {
			if types.Identical(templateDef.inputType, typeAny.Type()) {
				break
			}

			errMsg := fmt.Errorf("%w, template call expect expression", errTypeMismatch)
			err := parser.NewParseError(node.TemplateName, errMsg)
			errs = append(errs, err)
			return invalidTypes, inferences, errs
		}

		candidateType := expressionType[0]

		rhs := inferences.uniqueVariableInExpression
		if types.Identical(candidateType, typeAny.Type()) && rhs != nil {
			templateTree := newNodeImplicitType(
				templateName,
				templateDef.inputType,
				node.Range(),
			)
			recheck := newCollectionPostCheckImplicitTypeNode(
				rhs.candidate,
				templateTree,
				rhs.candidateDef,
				nil,
				rhs.candidateSymbol,
				nil,
			)
			recheck.operation = operatorCompatibleType

			inferences.variablesToRecheckAtEndOfScope = append(
				inferences.variablesToRecheckAtEndOfScope,
				recheck,
			)
			break
		}

		_, errMsg := TypeCheckCompatibilityWithConstraint(
			candidateType,
			templateDef.inputType,
		)
		if errMsg != nil {
			err := parser.NewParseError(node.TemplateName, errMsg)
			err.Range = node.Expression.Range()
			errs = append(errs, err)
			return invalidTypes, inferences, errs
		}

	default:
		panic(
			"'TemplateStatementNode' does not accept any type other than 'KindDefineTemplate', 'KindBlockTemplate', or 'KindUseTemplate'",
		)
	}

	return expressionType, inferences, errs
}

// definitionAnalysisComment parses go:code directives in comments to extract type hints
// and custom function definitions. Only one go:code per file is allowed.
//
//nolint:unparam // InferenceFoundReturn result is reserved for future use
func definitionAnalysisComment(
	comment *parser.CommentNode,
	parentScope *parser.GroupStatementNode,
	file *FileDefinition,
	_, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	var inferences InferenceFoundReturn

	commentType := [2]types.Type{
		typeAny.Type(),
		typeError.Type(),
	}

	if comment == nil || comment.Err != nil {
		return commentType, inferences, nil
	}

	if parentScope == nil {
		panic(
			"'CommentNode' cannot be parentless; it should be contained in at least one scope",
		)
	}

	if comment.Kind() != parser.KindComment {
		panic(
			"found value mismatch for 'CommentNode.Kind' during DefinitionAnalysis().\n " + comment.String(),
		)
	}

	if comment.GoCode == nil {
		return commentType, inferences, nil
	}

	// Do not analyze orphaned 'GoCode'
	// Correct 'GoCode' is available in parent scope
	if comment != parentScope.ShortCut.CommentGoCode {
		return commentType, inferences, nil
	}

	// 1. Find and store all functions and struct definitions
	const virtualFileName = "comment_for_go_template_virtual_file.go"
	const virtualHeader = "package main\n"

	fileSet := token.NewFileSet()
	source := append([]byte(virtualHeader), comment.GoCode.Value...)

	goNode, err := goParser.ParseFile(
		fileSet,
		virtualFileName,
		source,
		goParser.AllErrors,
	)

	var errsType []types.Error

	config := &types.Config{
		Importer:         importer.Default(),
		IgnoreFuncBodies: true,
		Error: func(err error) {
			errsType = append(errsType, func() types.Error {
				var target types.Error
				_ = errors.As(err, &target)
				return target
			}())
		},
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, _ := config.Check("", fileSet, []*ast.File{goNode}, info)
	var errComments []lexer.Error

	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)

		switch typ := obj.Type().(type) {
		case *types.Signature:
			function := &FunctionDefinition{}
			function.node = comment
			function.name = obj.Name()
			function.fileName = file.Name()
			function.typ = typ

			startPos := fileSet.Position(obj.Pos())
			endPos := fileSet.Position(obj.Pos())
			endPos.Column += endPos.Offset

			relativeRangeFunction := goAstPositionToRange(startPos, endPos)
			function.rng = remapRangeFromCommentGoCodeToSource(
				virtualHeader,
				comment.GoCode.Range,
				relativeRangeFunction,
			)

			if !parentScope.IsRoot() {
				err := parser.NewParseError(
					comment.GoCode,
					errors.New("function cannot be declared outside root scope"),
				)
				err.Range = function.Range()
				errComments = append(errComments, err)

				continue
			}

			file.functions[function.Name()] = function

		case *types.Named:

			if obj.Name() != "Input" {
				continue
			}

			if localVariables["."] == nil {
				continue
			}

			commentType[0] = typ

			convertGoAstPositionToProjectRange := func(goPosition token.Pos) lexer.Range {
				startPos := fileSet.Position(goPosition)
				endPos := fileSet.Position(goPosition)
				endPos.Column += endPos.Offset

				relativeRangeFunction := goAstPositionToRange(startPos, endPos)
				return remapRangeFromCommentGoCodeToSource(
					virtualHeader,
					comment.GoCode.Range,
					relativeRangeFunction,
				)
			}

			// No need to handle 'localVariables["$"]' since it ultimately point to 'localVariables["."]' anyway
			localVariables["."].rng = convertGoAstPositionToProjectRange(obj.Pos())
			localVariables["."].typ = typ.Underlying()

			rootNode := newNodeImplicitType(obj.Name(), typ, localVariables["."].rng)
			localVariables["."].TreeImplicitType = createImplicitTypeFromRealType(
				rootNode,
				convertGoAstPositionToProjectRange,
			)

		default:
			continue
		}
	}

	errs := convertThirdPartiesParseErrorToLocalError(
		err,
		errsType,
		file,
		comment,
		virtualHeader,
	)
	errs = append(errs, errComments...)

	return commentType, inferences, errs
}
