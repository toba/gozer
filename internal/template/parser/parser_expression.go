package parser

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/pacer/gozer/internal/template/lexer"
)

// parseVariableNames parses a list of variable names (e.g., "$a, $b").
// Returns the parsed variable tokens or an error with the range for reporting.
func (p *Parser) parseVariableNames() ([]*lexer.Token, *ParseError) {
	variables := make([]*lexer.Token, 0, maxVariablesPerDeclaration)

	count := 0
	for {
		count++

		if count > maxVariablesPerDeclaration {
			err := NewParseError(
				p.peek(),
				errors.New("only one or two variables can be declared at once"),
			)
			return variables, err
		}

		if !p.accept(lexer.DollarVariable) {
			err := NewParseError(
				p.peek(),
				errors.New("variable name must start with '$'"),
			)
			return variables, err
		}

		variables = append(variables, p.peek())
		p.nextToken()

		if p.expect(lexer.Comma) {
			continue
		}

		break
	}

	return variables, nil
}

// expressionStatementParser parses an expression, declaration, or assignment.
func (p *Parser) expressionStatementParser() (AstNode, *ParseError) {
	p.incRecursionDepth()
	defer p.decRecursionDepth()

	if multi, err := p.checkRecursionStatus(); err != nil {
		return multi, err
	}

outter_loop:
	for _, token := range p.stream.Tokens[p.indexCurrentToken:] {
		switch token.ID {
		case lexer.DeclarationAssignment:
			varDeclarationNode, err := p.declarationAssignmentParser()
			varDeclarationNode.Err = err

			return varDeclarationNode, err

		case lexer.Assignment:
			varInitialization, err := p.initializationAssignmentParser()
			varInitialization.Err = err

			return varInitialization, err

		case lexer.LeftParen:
			break outter_loop
		}
	}

	multiExpression, err := p.multiExpressionParser()
	multiExpression.Err = err

	return multiExpression, err
}

// declarationAssignmentParser parses "$var := expr" declarations.
//
//nolint:dupl // returns different type than initializationAssignmentParser
func (p *Parser) declarationAssignmentParser() (*VariableDeclarationNode, *ParseError) {
	if p.lastToken == nil {
		panic("unexpected empty token found at end of the current instruction")
	}

	varDeclarationNode := NewVariableDeclarationNode(
		KindVariableDeclaration,
		p.peek().Range.Start,
		p.lastToken.Range.End,
		nil,
	)

	variables, err := p.parseVariableNames()
	varDeclarationNode.VariableNames = variables
	if err != nil {
		err.Range = varDeclarationNode.rng
		varDeclarationNode.Err = err
		return varDeclarationNode, err
	}

	if !p.expect(lexer.DeclarationAssignment) {
		err := NewParseError(p.peek(), errors.New("expected assignment ':='"))
		varDeclarationNode.Err = err
		return varDeclarationNode, err
	}

	node, err := p.expressionStatementParser()
	expression, ok := node.(*MultiExpressionNode)

	if !ok {
		err := NewParseError(p.peek(), errors.New("expected an expression"))
		err.Range = node.Range()
		return varDeclarationNode, err
	}

	varDeclarationNode.Value = expression
	varDeclarationNode.Err = err

	if expression == nil {
		panic(
			"An AST, erroneous or not, must always be non <nil>. Can't be added to ControlFlow\n" + varDeclarationNode.String(),
		)
	}

	varDeclarationNode.rng.End = expression.Range().End

	return varDeclarationNode, err
}

// initializationAssignmentParser parses "$var = expr" assignments.
//
//nolint:dupl // returns different type than declarationAssignmentParser
func (p *Parser) initializationAssignmentParser() (*VariableAssignationNode, *ParseError) {
	if p.lastToken == nil {
		panic("unexpected empty token found at end of the current instruction")
	}

	varAssignation := NewVariableAssignmentNode(
		KindVariableAssignment,
		p.peek().Range.Start,
		p.lastToken.Range.End,
		nil,
	)

	variables, err := p.parseVariableNames()
	varAssignation.VariableNames = variables
	if err != nil {
		err.Range = varAssignation.rng
		varAssignation.Err = err
		return varAssignation, err
	}

	if !p.expect(lexer.Assignment) {
		err := NewParseError(p.peek(), errors.New("expected assignment '='"))
		varAssignation.Err = err
		return varAssignation, err
	}

	node, err := p.expressionStatementParser()
	expression, ok := node.(*MultiExpressionNode)

	if !ok {
		err := NewParseError(p.peek(), errors.New("expected an expression"))
		err.Range = node.Range()
		return varAssignation, err
	}

	varAssignation.Value = expression
	varAssignation.Err = err

	if expression == nil {
		panic(
			"An AST, erroneous or not, must always be non <nil>. Can't be added to ControlFlow\n" + varAssignation.String(),
		)
	}

	varAssignation.rng.End = expression.Range().End

	return varAssignation, err
}

// multiExpressionParser parses a pipeline of pipe-separated expressions.
func (p *Parser) multiExpressionParser() (*MultiExpressionNode, *ParseError) {
	if p.lastToken == nil {
		panic("unexpected empty token found at end of the current instruction")
	}

	multiExpression := NewMultiExpressionNode(
		KindMultiExpression,
		p.peek().Range.Start,
		p.lastToken.Range.End,
		nil,
	)

	var expression *ExpressionNode
	var err *ParseError

	for next := true; next; next = p.expect(lexer.Pipe) {
		expression, err = p.expressionParser() // main parsing
		expression.Err = err

		multiExpression.Expressions = append(multiExpression.Expressions, expression)

		if err != nil {
			multiExpression.Err = err
			return multiExpression, err
		}

		if expression == nil { // because if err == nil, then expression != nil
			panic(
				"returned AST was nil although parsing completed successfully. can't be added to ControlFlow\n" + multiExpression.String(),
			)
		}
	}

	multiExpression.rng.End = expression.Range().End

	return multiExpression, nil
}

// expressionParser parses a single expression (function call, variable access, literal, etc.).
func (p *Parser) expressionParser() (*ExpressionNode, *ParseError) {
	if p.lastToken == nil {
		panic("unexpected empty token found at end of the current instruction")
	}

	expression := NewExpressionNode(KindExpression, p.peek().Range)
	expression.rng.End = p.lastToken.Range.End

	expression.ExpandedTokens = make([]AstNode, 0, p.sizeStream)
	expression.Symbols = make([]*lexer.Token, 0, p.sizeStream)

	count := 0
	for p.accept(lexer.Function) || p.accept(lexer.DotVariable) || p.accept(lexer.DollarVariable) || p.accept(lexer.StringLit) || p.accept(lexer.Character) ||
		p.accept(lexer.LeftParen) || p.accept(lexer.RightParen) || p.accept(lexer.Number) || p.accept(lexer.Decimal) || p.accept(lexer.ComplexNumber) || p.accept(lexer.Boolean) {
		count++
		if count > maxExpressionTokens {
			panic("possible infinite loop detected while parsing 'expression'")
		}

		if p.accept(lexer.LeftParen) {
			leftParenthesis := p.peek()
			p.nextToken() // skip '('

			node, err := p.expressionStatementParser() // main processing
			node.SetError(err)

			tokenName := "$__expandable_token_" + strconv.Itoa(GetUniqueNumber())
			group := lexer.NewToken(
				lexer.ExpandableGroup,
				leftParenthesis.Range,
				[]byte(tokenName),
			)
			group.Range.End = node.Range().End

			expression.ExpandedTokens = append(expression.ExpandedTokens, node)
			expression.Symbols = append(expression.Symbols, group)

			if err != nil {
				return expression, err
			}

			rightParenthesis := p.peek()

			if !p.accept(lexer.RightParen) {
				err := NewParseError(
					rightParenthesis,
					errors.New("missing closing parenthesis ')'"),
				)
				expression.SetError(err)
				return expression, err
			}

			group.Range.End = rightParenthesis.Range.End
			p.nextToken() // skip ')'

			// 2. handle case where: (expression).field VS (expression) .field
			if next := p.peek(); next != nil {
				distance := next.Range.Start.Character - (rightParenthesis.Range.End.Character - 1)

				// check whether the 'next' token is right next to parenthesis
				if distance == 1 && next.ID == lexer.DotVariable {
					group.Range.End = next.Range.End
					group.Value = append(group.Value, next.Value...)
					p.nextToken() // skip 'dot var'

					if len(string(next.Value)) == 1 {
						err := NewParseError(
							group,
							errors.New("sub-expression cannot end with '.'"),
						)
						expression.SetError(err)
						return expression, err
					}
				} else if distance == 1 && next.ID == lexer.RightParen {
					// do nothing, bc there is no error to report
				} else if distance == 1 {
					err := NewParseError(next, errors.New("need space between argument"))
					expression.SetError(err)
					return expression, err
				}
			}

			continue
		} else if p.accept(lexer.RightParen) {
			break
		}

		expression.ExpandedTokens = append(expression.ExpandedTokens, nil)
		expression.Symbols = append(expression.Symbols, p.peek())
		p.nextToken()
	}

	if len(expression.Symbols) != len(expression.ExpandedTokens) {
		panic(
			fmt.Sprintf(
				"length mismatch between Symbols (%d) and ExpandedTokens (%d). stream = %s",
				len(expression.Symbols),
				len(expression.ExpandedTokens),
				p.stream.String(),
			),
		)
	}

	if len(expression.Symbols) == 0 {
		err := NewParseError(p.peek(), errors.New("empty expression"))
		err.Range.Start = p.peek().Range.End
		err.Range.Start.Character--
		expression.SetError(err)
		return expression, err
	}

	size := len(expression.Symbols)
	expression.rng.End = expression.Symbols[size-1].Range.End
	// expression.rng.End = p.peek().Range.End

	return expression, nil
}
