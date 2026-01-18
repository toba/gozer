package analyzer

import (
	"fmt"
	"go/types"
	"log"
	"strconv"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

func definitionAnalysisMultiExpression(
	node *parser.MultiExpressionNode,
	parent *parser.GroupStatementNode,
	file *FileDefinition,
	globalVariables, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	if node.Kind() != parser.KindMultiExpression {
		panic(
			"found value mismatch for 'MultiExpressionNode.Kind' during DefinitionAnalysis()\n" + node.String(),
		)
	}

	var errs, localErrs []lexer.Error
	var inferences, localInferences InferenceFoundReturn
	expressionType := [2]types.Type{
		types.Typ[types.Invalid],
		typeError.Type(),
	}

	if node.Err != nil {
		return expressionType, localInferences, nil
	}

	for count, expression := range node.Expressions {
		if expression == nil {
			log.Printf(
				"fatal, nil element within expression list for MultiExpressionNode. \n %s \n",
				node.String(),
			)
			panic(
				"element within expression list cannot be 'nil' for MultiExpressionNode. instead of inserting the nil value, omit it",
			)
		}

		// normal processing when this is the first expression
		if count == 0 {
			expressionType, localInferences, localErrs = definitionAnalysisExpression(
				expression,
				parent,
				file,
				globalVariables,
				localVariables,
			)

			errs = append(errs, localErrs...)
			inferences = localInferences
			continue
		}

		// when piping, you pass the result of the previous expression to the end position of the current expression
		//
		// create a token group and insert it to the end of the expression
		groupName := nameTempVar + "_GROUP_" + strconv.Itoa(parser.GetUniqueNumber())
		tokenGroup := &lexer.Token{
			ID:    lexer.StaticGroup,
			Value: []byte(groupName),
			Range: expression.Range(),
		}
		expression.Symbols = append(expression.Symbols, tokenGroup)

		// then insert that token as variable within the file
		def := NewVariableDefinition(groupName, nil, parent, file.Name())
		def.rng = tokenGroup.Range

		def.typ = expressionType[0]
		if localInferences.uniqueVariableInExpression != nil {
			def.TreeImplicitType = localInferences.uniqueVariableInExpression.candidate
		}

		localVariables[def.Name()] = def
		expressionType, localInferences, localErrs = definitionAnalysisExpression(
			expression,
			parent,
			file,
			globalVariables,
			localVariables,
		)

		errs = append(errs, localErrs...)
		inferences.uniqueVariableInExpression = localInferences.uniqueVariableInExpression
		inferences.variablesToRecheckAtEndOfScope = append(
			inferences.variablesToRecheckAtEndOfScope,
			localInferences.variablesToRecheckAtEndOfScope...)

		// once, processing is over, remove the group created from the expression
		size := len(expression.Symbols)
		expression.Symbols = expression.Symbols[:size-1]

		delete(localVariables, def.Name())
	}

	return expressionType, inferences, errs
}

func definitionAnalysisExpression(
	node *parser.ExpressionNode,
	parent *parser.GroupStatementNode,
	file *FileDefinition,
	globalVariables, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	if node.Kind() != parser.KindExpression {
		panic(
			"found value mismatch for 'ExpressionNode.Kind'; expected 'KindExpression' instead. Current node:\n" + node.String(),
		)
	}

	if globalVariables == nil || localVariables == nil {
		panic(
			"'globalVariables' or 'localVariables' or shouldn't be empty for 'ExpressionNode.DefinitionAnalysis()'",
		)
	}

	var expressionType = [2]types.Type{types.Typ[types.Invalid], typeError.Type()}
	var errs []lexer.Error
	inferences := InferenceFoundReturn{}

	if node.Err != nil {
		return expressionType, inferences, nil
	}

	if len(node.Symbols) == 0 {
		err := parser.NewParseError(&lexer.Token{}, errEmptyExpression)
		err.Range = node.Range()
		errs = append(errs, err)
		return expressionType, inferences, errs
	}

	defChecker := NewDefinitionAnalyzer(
		node.Symbols,
		node.ExpandedTokens,
		parent,
		file,
		node.Range(),
	)
	expressionType, inferences, errs = defChecker.makeSymboleDefinitionAnalysis(
		localVariables,
		globalVariables,
	)

	if expressionType[0] == nil {
		log.Printf(
			"found a <nil> return type for expression"+"\n file = %#v\n node = %#v\n inferences = %#v\n",
			file,
			node,
			inferences,
		)
		panic("found a <nil> return type for expression")
	}

	return expressionType, inferences, errs
}

// first make definition analysis to find all existing reference
// then make the type analysis
type definitionAnalyzer struct {
	symbols         []*lexer.Token
	expandedTokens  []parser.AstNode
	index           int // current index of symbols within the expression
	isEOF           bool
	parent          *parser.GroupStatementNode
	file            *FileDefinition
	rangeExpression lexer.Range
}

func NewDefinitionAnalyzer(
	symbols []*lexer.Token,
	expandedGroups []parser.AstNode,
	parent *parser.GroupStatementNode,
	file *FileDefinition,
	rangeExpr lexer.Range,
) *definitionAnalyzer {
	ana := &definitionAnalyzer{
		symbols:         symbols,
		index:           0,
		parent:          parent,
		file:            file,
		rangeExpression: rangeExpr,
		expandedTokens:  expandedGroups,
	}

	return ana
}

func (a definitionAnalyzer) String() string {
	str := fmt.Sprintf(
		`{ "symbols": %s, "index": %d, "file": %v, "rangeExpression": %s }`,
		lexer.PrettyFormater(a.symbols),
		a.index,
		a.file,
		a.rangeExpression.String(),
	)
	return str
}

func (a *definitionAnalyzer) peek() *lexer.Token {
	if a.index >= len(a.symbols) {
		panic(
			"index out of bound for 'definitionAnalyzer'; check that you only use the provide method to move between tokens, like 'analyzer.nextToken()'",
		)
	}

	return a.symbols[a.index]
}

func (a *definitionAnalyzer) nextToken() {
	if a.index+1 >= len(a.symbols) {
		a.isEOF = true
		return
	}

	a.index++
}

func (a *definitionAnalyzer) isTokenAvailable() bool {
	if a.isEOF {
		return false
	}

	return a.index < len(a.symbols)
}

// fetch all tokens and sort them
func (p *definitionAnalyzer) makeSymboleDefinitionAnalysis(
	localVariables, globalVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	var errs []lexer.Error
	var err *parser.ParseError
	lateVariableRecheck := InferenceFoundReturn{}

	processedToken := []*lexer.Token{}
	processedTypes := []types.Type{}

	makeTypeInference := func(symbol *lexer.Token, symbolType, constraintType types.Type) (*collectionPostCheckImplicitTypeNode, *parser.ParseError) {
		return makeTypeInferenceWhenPossible(
			symbol,
			symbolType,
			constraintType,
			localVariables,
			globalVariables,
		)
	}

	var lastVarSymbol *lexer.Token
	var symbolType types.Type
	var varDef *VariableDefinition

	count := 0

	for p.isTokenAvailable() {
		if count > 100 {
			log.Printf("loop took too long to complete.\n analyzer = %s\n", p)
			panic("loop lasted more than expected on 'expression definition analysis'")
		}

		count++
		symbol := p.peek()

		{ // temporary scope to avoid namespace pollution
			// the goal of this is to make 'key' value of the loop capable of type inference (for ease of use by the human user)
			def := p.file.extraVariableNameWithTypeInferenceBehavior[string(symbol.Value)]

			if def != nil && symbol.ID == lexer.DollarVariable {
				foundDef, _ := getVariableDefinitionForRootField(
					symbol,
					localVariables,
					globalVariables,
				)

				if def == foundDef {
					symbol = lexer.CloneToken(symbol)
					symbol.ID = lexer.DotVariable
				}
			}
		}

		switch symbol.ID {
		case lexer.StringLit:
			processedToken = append(processedToken, symbol)
			processedTypes = append(
				processedTypes,
				mustGetBasicTypeFromTokenID(symbol.ID),
			) // String

			p.nextToken()
		case lexer.Character:
			processedToken = append(processedToken, symbol)
			processedTypes = append(
				processedTypes,
				mustGetBasicTypeFromTokenID(symbol.ID),
			) // Rune

			p.nextToken()
		case lexer.Number:
			processedToken = append(processedToken, symbol)
			processedTypes = append(
				processedTypes,
				mustGetBasicTypeFromTokenID(symbol.ID),
			) // Int

			p.nextToken()
		case lexer.Decimal:
			processedToken = append(processedToken, symbol)
			processedTypes = append(
				processedTypes,
				mustGetBasicTypeFromTokenID(symbol.ID),
			) // Float64

			p.nextToken()
		case lexer.ComplexNumber:
			processedToken = append(processedToken, symbol)
			processedTypes = append(
				processedTypes,
				mustGetBasicTypeFromTokenID(symbol.ID),
			) // Complex64

			p.nextToken()
		case lexer.Boolean:
			processedToken = append(processedToken, symbol)
			processedTypes = append(
				processedTypes,
				mustGetBasicTypeFromTokenID(symbol.ID),
			) // Bool

			p.nextToken()
		case lexer.Function:
			var fakeVarDef *VariableDefinition
			fields, _, _, _ := splitVariableNameFields(symbol)

			functionName := fields[0]
			def := p.file.functions[functionName]

			if def == nil {
				symbolType = types.Typ[types.Invalid]
				err = parser.NewParseError(symbol, errFunctionUndefined)
				err.Range.End.Character = err.Range.Start.Character + len(functionName)
				errs = append(errs, err)
			} else {
				fakeVarDef = NewVariableDefinition(
					def.name,
					def.node,
					p.parent,
					def.fileName,
				)
				fakeVarDef.typ = def.typ

				symbolType, err = getRealTypeAssociatedToVariable(symbol, fakeVarDef)
				if err != nil {
					errs = append(errs, err)
				}
			}

			processedToken = append(processedToken, symbol)
			processedTypes = append(processedTypes, symbolType)

			p.nextToken()

		case lexer.ExpandableGroup:

			node := p.expandedTokens[p.index]
			if node == nil {
				panic("no associated AST found for 'expanded_token' " + symbol.String())
			}

			typs, inferences, localErrs := definitionAnalysisRecursive(
				node,
				p.parent,
				p.file,
				globalVariables,
				localVariables,
			)
			errs = append(errs, localErrs...)

			// symbol = lexer.CloneToken(symbol)
			fields, _, _, err := splitVariableNameFields(symbol)
			if err != nil { // also an error when len(fields) == 0
				panic("malformated symbol name for 'expanded_token'. " + err.GetError())
			}

			varName := fields[0]
			varDef = NewVariableDefinition(varName, node, p.parent, p.file.name)
			varDef.typ = typs[0]

			localVariables[varName] = varDef

			if types.Identical(typs[0], typeAny.Type()) &&
				inferences.uniqueVariableInExpression != nil {
				rhs := inferences.uniqueVariableInExpression
				varDef.TreeImplicitType = rhs.candidate
			}

			fallthrough

		case lexer.DollarVariable, lexer.StaticGroup:
			lastVarSymbol = symbol
			symbolType = typeAny.Type()

			varDef, err = getVariableDefinitionForRootField(
				symbol,
				localVariables,
				globalVariables,
			)

			if err != nil {
				errs = append(errs, err)
				symbolType = types.Typ[types.Invalid]
			} else if !types.Identical(varDef.typ, typeAny.Type()) {
				symbolType, err = getRealTypeAssociatedToVariable(symbol, varDef)
				if err != nil {
					errs = append(errs, err)
				}

				// NOTE: this is essential to test if a field **exist** when using '$' variable
				// type check will always pass, but this help in verifying that 'varTree.toDiscard != true' and type syst. defeated
			} else if types.Identical(varDef.typ, typeAny.Type()) {
				varTree := extractOrInsertTemporaryImplicitTypeFromVariable(
					varDef,
					symbol,
				)
				varFakeName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
				fakeTree := newNodeImplicitType(
					varFakeName,
					typeAny.Type(),
					symbol.Range,
				) // read note above

				recheck := newCollectionPostCheckImplicitTypeNode(
					varTree,
					fakeTree,
					varDef,
					nil,
					symbol,
					nil,
				)
				lateVariableRecheck.variablesToRecheckAtEndOfScope = append(
					lateVariableRecheck.variablesToRecheckAtEndOfScope,
					recheck,
				)
				symbolType = typeAny.Type() // this help pushback processing in the next step
			}

			/*
				symbolType, err = getRealTypeAssociatedToVariable(symbol, varDef)
				if err != nil && !errors.Is(err.Err, errDefeatedTypeSystem) {
					errs = append(errs, err)
				}
			*/

			markVariableAsUsed(varDef)

			processedToken = append(processedToken, symbol)
			processedTypes = append(processedTypes, symbolType)

			p.nextToken()

		case lexer.DotVariable:

			lastVarSymbol = symbol
			symbolType = typeAny.Type()

			varDef, err = getVariableDefinitionForRootField(
				symbol,
				localVariables,
				globalVariables,
			)

			if err != nil {
				errs = append(errs, err)
				symbolType = types.Typ[types.Invalid]
			} else if !types.Identical(varDef.typ, typeAny.Type()) {
				symbolType, err = getRealTypeAssociatedToVariable(symbol, varDef)
				if err != nil {
					errs = append(errs, err)
				}
			} else if types.Identical(varDef.typ, typeAny.Type()) {
				symbolType, err = updateVariableImplicitType(varDef, symbol, symbolType)
				if err != nil {
					errs = append(errs, err)
				}
			}

			/*
				symbolType, err = getRealTypeAssociatedToVariable(symbol, varDef)
				if err != nil && !errors.Is(err.Err, errDefeatedTypeSystem) {
					errs = append(errs, err)
				}
			*/

			markVariableAsUsed(varDef)

			processedToken = append(processedToken, symbol)
			processedTypes = append(processedTypes, symbolType)

			p.nextToken()

		default: // LeftParen, RightParen, etc.
			panic(
				"unexpected token type during 'symbol analysis' on Expression. " + symbol.String(),
			)
		}
	}

	// Only necessary to pass around single symbol expression to other node
	if count == 1 && varDef != nil && lastVarSymbol != nil {
		varFakeName := nameTempVar + strconv.Itoa(parser.GetUniqueNumber())
		fakeTree := newNodeImplicitType(varFakeName, symbolType, lastVarSymbol.Range)
		varTree := extractOrInsertTemporaryImplicitTypeFromVariable(varDef, lastVarSymbol)

		recheck := newCollectionPostCheckImplicitTypeNode(
			varTree,
			fakeTree,
			varDef,
			nil,
			lastVarSymbol,
			nil,
		)
		lateVariableRecheck.uniqueVariableInExpression = recheck
	}

	groupType, variablesToRecheck, localErrs := makeExpressionTypeCheck(
		processedToken,
		processedTypes,
		makeTypeInference,
		p.rangeExpression,
	)
	for _, err := range localErrs {
		errs = append(errs, err)
	}

	lateVariableRecheck.variablesToRecheckAtEndOfScope = append(
		lateVariableRecheck.variablesToRecheckAtEndOfScope,
		variablesToRecheck...)

	return groupType, lateVariableRecheck, errs
}
