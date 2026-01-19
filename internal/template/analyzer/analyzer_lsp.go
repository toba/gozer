package analyzer

import (
	"fmt"
	"go/types"
	"log"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

// ------------
// ------------
// LSP-like helper functions
// ------------
// ------------

// FindSourceDefinitionFromPosition returns the NodeDefinition(s) at the given position.
// NodeDefinition can be FileDefinition, VariableDefinition, TemplateDefinition, or FunctionDefinition.
func FindSourceDefinitionFromPosition(
	file *FileDefinition,
	position lexer.Position,
) []NodeDefinition {
	// 1. Find the node and token corresponding to the provided position
	seeker := &findAstNodeRelatedToPosition{Position: position, fileName: file.name}

	log.Println("position before walker: ", position)
	parser.Walk(seeker, file.root)
	log.Printf("seeker after walker : %#v\n", seeker)

	if seeker.TokenFound == nil { // No definition found
		return nil
	}

	//
	// 2. From the node and token found, find the appropriate 'Source Definition'
	//
	invalidVariableDefinition := NewVariableDefinition(
		string(seeker.TokenFound.Value),
		seeker.NodeFound,
		seeker.LastParent,
		file.FileName(),
	)
	invalidVariableDefinition.typ = types.Typ[types.Invalid]
	invalidVariableDefinition.rng.Start = position

	if seeker.IsTemplate {
		var allTemplateDefs []NodeDefinition = nil
		TemplateManager := TemplateManager

		templateName := string(seeker.TokenFound.Value)

		for templateScope, def := range TemplateManager.TemplateScopeToDefinition {
			if templateName == templateScope.TemplateName() {
				allTemplateDefs = append(allTemplateDefs, def)
			}
		}

		return allTemplateDefs
	} else if seeker.IsExpression || seeker.IsVariable {
		// handle case where 'seeker.tokenFound' is either 'string', 'number', 'bool'
		basicDefinition := createNodeDefinitionForBasicType(
			seeker.TokenFound,
			seeker.NodeFound,
			file.name,
		)
		if basicDefinition != nil {
			return []NodeDefinition{basicDefinition}
		}

		fields, _, fieldPosCountedFromBack, errSplit := splitVariableNameFields(
			seeker.TokenFound,
		)
		if errSplit != nil {
			log.Printf(
				"warning, spliting variable name was unsuccessful :: varName = %s\n",
				string(seeker.TokenFound.Value),
			)
			singleDefinition := []NodeDefinition{invalidVariableDefinition}
			return singleDefinition
		}

		if len(fields) == 0 {
			return nil
		}

		var symbolDefinition NodeDefinition
		rootVarName := fields[0]

		// Check whether it is a function or variable
		functionDef := file.functions[rootVarName]
		variableDef := file.GetVariableDefinitionWithinScope(
			rootVarName,
			seeker.LastParent,
		)

		if functionDef != nil {
			symbolDefinition = functionDef
		} else if variableDef != nil {
			symbolDefinition = variableDef
		} else {
			singleDefinition := []NodeDefinition{invalidVariableDefinition}
			return singleDefinition
		}

		if len(fields) == 1 {
			singleDefinition := []NodeDefinition{symbolDefinition}
			return singleDefinition
		}

		relativeCursorPosition := position
		relativeCursorPosition.Line = position.Line - seeker.TokenFound.Range.End.Line
		relativeCursorPosition.Character = (seeker.TokenFound.Range.End.Character - 1) - position.Character // char > 0

		fieldIndex := findFieldContainingRange(
			fieldPosCountedFromBack,
			relativeCursorPosition,
		)

		newVarName, err := joinVariableNameFields(fields[:fieldIndex+1])
		if err != nil {
			log.Printf(
				"variable name was split successfully, but now cannot be joined for some reason"+
					"\n fields = %q\n",
				fields,
			)
			panic(
				"variable name was split successfully, but now cannot be joined for some reason",
			)
		}

		newToken := lexer.CloneToken(seeker.TokenFound)
		newToken.Value = []byte(newVarName)
		newToken.Range.End.Character = newToken.Range.End.Character - fieldPosCountedFromBack[fieldIndex] + len(
			fields[fieldIndex],
		)

		temporaryVariableDef := NewVariableDefinition(
			symbolDefinition.Name(),
			symbolDefinition.Node(),
			nil,
			symbolDefinition.FileName(),
		)
		temporaryVariableDef.typ = symbolDefinition.Type()

		typ, err := getRealTypeAssociatedToVariable(newToken, temporaryVariableDef)
		if err != nil {
			log.Printf("error while analysis variable chain :: "+err.String()+
				"\n\n associated type = %s\n", typ)
		}

		variableDef = NewVariableDefinition(
			newVarName,
			seeker.NodeFound,
			seeker.LastParent,
			file.FileName(),
		)
		variableDef.typ = typ

		variableDef.rng = newToken.Range

		varDef, ok := symbolDefinition.(*VariableDefinition)
		if ok && varDef.TreeImplicitType != nil {
			reach := getVariableImplicitRange(varDef, newToken)
			if reach != nil {
				variableDef.rng = *reach
			}
		}

		if fieldIndex == 0 {
			variableDef.rng = symbolDefinition.Range()
		}

		singleDefinition := []NodeDefinition{variableDef}

		return singleDefinition
	} else if seeker.IsKeyword {
		switch node := seeker.NodeFound.(type) {
		case *parser.SpecialCommandNode:
			target := node.Target
			if target == nil {
				return nil
			}

			def := NewKeywordSymbolDefinition(
				target.StreamToken.String(),
				file.name,
				target,
			)
			singleDefinition := []NodeDefinition{def}

			return singleDefinition

		case *parser.GroupStatementNode:
			target := node.NextLinkedSibling
			if target == nil {
				return nil
			}

			def := NewKeywordSymbolDefinition(
				target.StreamToken.String(),
				file.name,
				target,
			)
			singleDefinition := []NodeDefinition{def}

			return singleDefinition

		default:
		}

		panic("keyword symbol definition finder not implemented yet!")
	}

	return nil
}

func createNodeDefinitionForBasicType(
	token *lexer.Token,
	node parser.AstNode,
	fileName string,
) NodeDefinition {
	def := &BasicSymbolDefinition{
		node:     node,
		rng:      token.Range,
		fileName: fileName,
		name:     string(token.Value),
		typ:      getBasicTypeFromTokenID(token.ID),
	}

	if def.typ == nil {
		return nil
	}

	if token.ID == lexer.StringLit {
		def.name = fmt.Sprintf("`%s`", string(token.Value))
	}

	return def
}

func mustGetBasicTypeFromTokenID(tokenId lexer.Kind) *types.Basic {
	typ := getBasicTypeFromTokenID(tokenId)
	if typ == nil {
		panic("no basic type found for this token kind: " + tokenId.String())
	}

	return typ
}

func getBasicTypeFromTokenID(tokenId lexer.Kind) *types.Basic {
	switch tokenId {
	case lexer.Number:
		return types.Typ[types.Int]
	case lexer.Decimal:
		return types.Typ[types.Float64]
	case lexer.ComplexNumber:
		return types.Typ[types.Complex128]
	case lexer.Boolean:
		return types.Typ[types.Bool]
	case lexer.StringLit:
		return types.Typ[types.String]
	case lexer.Character:
		return types.Typ[types.Int]
	}

	return nil
}

type findAstNodeRelatedToPosition struct {
	Position     lexer.Position
	TokenFound   *lexer.Token
	LastParent   *parser.GroupStatementNode
	NodeFound    parser.AstNode // nodeStatement
	fileName     string
	IsTemplate   bool
	IsVariable   bool
	IsExpression bool
	IsKeyword    bool
	IsHeader     bool
}

func (v *findAstNodeRelatedToPosition) SetHeaderFlag(ok bool) {
	v.IsHeader = ok
}

// the search for the appropriate node is highly dependent on AstNode 'Range'
// If the parent node do not contain the range of its children, then they will never be discover/found
func (v *findAstNodeRelatedToPosition) Visit(node parser.AstNode) parser.Visitor {
	if node == nil {
		return nil
	}

	// 1. Going down the node tree
	if v.TokenFound != nil {
		return nil
	}

	if !node.Range().Contains(v.Position) {
		return nil
	}

	switch n := node.(type) {
	case *parser.GroupStatementNode:
		if n.KeywordRange.Contains(v.Position) {
			v.TokenFound = n.KeywordToken
			v.NodeFound = n
			v.IsKeyword = true

			return nil
		}

		v.LastParent = n
		return v
	case *parser.TemplateStatementNode:
		if n.TemplateName == nil {
			return nil
		} else if n.TemplateName.Range.Contains(v.Position) {
			v.TokenFound = n.TemplateName
			v.NodeFound = n
			v.IsTemplate = true
		}

		v.NodeFound = n

		return v
	case *parser.VariableAssignationNode:
		for _, variable := range n.VariableNames {
			if !variable.Range.Contains(v.Position) {
				continue
			}
			v.TokenFound = variable
			v.NodeFound = n
			v.IsVariable = true

			if v.IsHeader {
				v.LastParent = v.LastParent.Parent()
			}

			return nil
		}

		v.NodeFound = n

		return v
	case *parser.VariableDeclarationNode:
		for _, variable := range n.VariableNames {
			if variable.Range.Contains(v.Position) {
				v.TokenFound = variable
				v.NodeFound = n
				v.IsVariable = true

				return nil
			}
		}

		v.NodeFound = n

		return v
	case *parser.MultiExpressionNode:
		for _, expression := range n.Expressions {
			if expression.Range().Contains(v.Position) {
				return v
			}
		}

		return nil
	case *parser.ExpressionNode:
		for index, symbol := range n.Symbols {
			if !symbol.Range.Contains(v.Position) {
				continue
			}
			// if expendable token, trigger analysis of all 'ExpandedTokens' element
			if symbol.ID == lexer.ExpandableGroup {
				childExpression := n.ExpandedTokens[index]
				if childExpression == nil {
					panic(
						"no AST Node found for expandable token at " + symbol.Range.String() + " in " + v.fileName,
					)
				}

				if childExpression.Range().Contains(v.Position) {
					return v
				}

				// if we reach here, this mean that the token is expandable
				// but we are either targeting the paren '(' or ')'
				// or even the 'fields' connected to the root var name
			}

			// otherwise
			v.TokenFound = symbol
			v.IsExpression = true

			if v.NodeFound == nil {
				v.NodeFound = n
			}

			if v.IsHeader {
				v.LastParent = v.LastParent.Parent()
			}

			return nil
		}

	case *parser.SpecialCommandNode:
		if n.Range().Contains(v.Position) {
			v.TokenFound = n.Value
			v.NodeFound = n
			v.IsKeyword = true

			return nil
		}
	}

	return nil
}

// GoToDefinition finds the definition location for a given token.
func GoToDefinition(
	from *lexer.Token,
	parentNodeStatement parser.AstNode,
	parentScope *parser.GroupStatementNode,
	file *FileDefinition,
	isTemplate bool,
) (fileName string, defFound parser.AstNode, reach lexer.Range) {
	if file == nil {
		log.Printf(
			"File definition not found to compute Go-To Definition. Thus cannot find definition of node."+
				"parentScope = %s\n parentNodeStatement = %s \n from = %s\n",
			parentScope,
			parentNodeStatement,
			from,
		)
		panic(
			"File definition not found to compute Go-To Definition. Thus cannot find definition of node, " + from.String(),
		)
	}
	// 1. Try to find the template, if appropriate
	if isTemplate {
		templateFound := file.templates[string(from.Value)]

		if templateFound == nil {
			return file.Name(), nil, lexer.Range{}
		}

		if templateFound.fileName == "" {
			return file.Name(), nil, lexer.Range{}
		}

		return templateFound.fileName, templateFound.Node(), templateFound.Range()
	}

	name := string(from.Value)

	// 2. Try to find the function
	functionFound := file.functions[name]
	if functionFound != nil {
		return functionFound.fileName, functionFound.Node(), functionFound.Range()
	}

	// 3. Found the multi-scope varialbe
	var count = 0

	// Bubble up until you find the scope where the variable is defined
	for parentScope != nil {
		count++
		if count > MaxLoopRepetition {
			panic("possible infinite loop detected while processing 'goToDefinition()'")
		}

		scopedVariables := file.GetScopedVariables(parentScope)

		variableFound, ok := scopedVariables[name]
		if ok {
			return variableFound.fileName, variableFound.Node(), variableFound.Range()
		}

		parentScope = parentScope.Parent()
	}

	return file.Name(), nil, lexer.Range{}
}

func Hover(definition NodeDefinition) (string, lexer.Range) {
	if definition == nil {
		panic("Hover() does not accept <nil> definition")
	}

	reach := definition.Range()
	typeStringified := definition.TypeString()
	typeStringified = "```go\n" + typeStringified + "\n```"

	return typeStringified, reach
}
