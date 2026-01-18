package parser

import (
	"errors"

	"github.com/pacer/gozer/internal/template/lexer"
)

// ParseStatement parses a statement from tokens.
func (p *Parser) ParseStatement() (ast AstNode, er *ParseError) {
	if p.stream.IsEmpty() {
		err := NewParseError(p.peek(), errors.New("empty statement"))
		multiExpression := NewMultiExpressionNode(
			KindMultiExpression,
			p.peek().Range.Start,
			p.peek().Range.End,
			err,
		)
		return multiExpression, err
	}

	// 1. Escape infinite recursion
	p.incRecursionDepth()
	defer p.decRecursionDepth()

	if multi, err := p.checkRecursionStatus(); err != nil {
		return multi, err
	}

	if p.lastToken == nil {
		panic("unexpected empty token found at end of the current instruction")
	}

	// 3. Syntax Parser for the Go Template language
	if p.accept(lexer.Keyword) {
		keywordToken := p.peek()

		if handler, ok := keywordHandlers[string(keywordToken.Value)]; ok {
			return handler(p, keywordToken)
		}
	} else if p.accept(lexer.Comment) {
		commentExpression := &CommentNode{
			kind:  KindComment,
			Value: p.peek(),
			rng:   p.peek().Range,
		}

		p.nextToken()

		if !p.expect(lexer.Eol) {
			err := NewParseError(
				p.peek(),
				errors.New(
					"syntax for comment didn't end properly. extraneous expression",
				),
			)
			err.Range.End = p.lastToken.Range.End
			commentExpression.Err = err
			return commentExpression, err
		}

		// Check that this comment contains go code to semantically analize
		lookForAndSetGoCodeInComment(commentExpression)

		return commentExpression, nil
	}

	// 4. Default parser whenever no other parser have been enabled
	expression, err := p.expressionStatementParser()

	if err != nil {
		expression.SetError(err)
		return expression, err
	}

	if !p.expect(lexer.Eol) {
		err = NewParseError(p.peek(), errors.New("expected end of statement"))
		err.Range.End = p.lastToken.Range.End
		expression.SetError(err)
		return expression, err
	}

	return expression, nil
}
