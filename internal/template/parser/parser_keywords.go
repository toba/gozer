package parser

import (
	"errors"

	"github.com/pacer/gozer/internal/template/lexer"
)

// keywordHandler is a function that parses a specific keyword.
type keywordHandler func(p *Parser, keywordToken *lexer.Token) (AstNode, *ParseError)

// keywordHandlers maps keyword strings to their parse handlers.
// Initialized in init() to avoid initialization cycle.
var keywordHandlers map[string]keywordHandler

func init() {
	keywordHandlers = map[string]keywordHandler{
		"if":       (*Parser).parseIfKeyword,
		"else":     (*Parser).parseElseKeyword,
		"range":    (*Parser).parseRangeKeyword,
		"with":     (*Parser).parseWithKeyword,
		"block":    (*Parser).parseBlockKeyword,
		"define":   (*Parser).parseDefineKeyword,
		"template": (*Parser).parseTemplateKeyword,
		"end":      (*Parser).parseEndKeyword,
		"break":    (*Parser).parseBreakKeyword,
		"continue": (*Parser).parseContinueKeyword,
	}
}

// parseIfKeyword handles the "if" keyword.
func (p *Parser) parseIfKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	ifExpression := NewGroupStatementNode(KindIf, keywordToken.Range, p.stream)
	ifExpression.rng.End = p.lastToken.Range.End
	ifExpression.KeywordRange = keywordToken.Range
	ifExpression.KeywordToken = keywordToken

	p.nextToken() // skip keyword "if"

	expression, err := p.ParseStatement()

	ifExpression.ControlFlow = expression
	ifExpression.Err = err

	ifExpression.rng.End = expression.Range().End

	if err != nil {
		return ifExpression, err
	}

	if expression == nil { // because if err == nil, then expression != nil
		panic(
			"returned AST was nil although parsing completed successfully. can't be added to ControlFlow\n" + ifExpression.String(),
		)
	}

	// Special case: "else" keyword after "if" suggests user meant "else if"
	if expression.Kind() == KindElse {
		err = NewParseError(
			&lexer.Token{},
			errors.New("did you mean 'else if'?"),
		)
		err.Range = expression.Range()
		err.Range.Start = ifExpression.rng.Start
		ifExpression.Err = err
		return ifExpression, err
	}

	if err = validateControlFlowExpression(expression, "if"); err != nil {
		ifExpression.Err = err
		return ifExpression, err
	}

	return ifExpression, nil
}

// parseElseKeyword handles the "else" keyword.
func (p *Parser) parseElseKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	elseExpression := NewGroupStatementNode(
		KindElse,
		keywordToken.Range,
		p.stream,
	)
	elseExpression.rng.End = p.lastToken.Range.End
	elseExpression.KeywordRange = keywordToken.Range
	elseExpression.KeywordToken = keywordToken

	p.nextToken() // skip 'else' token

	if p.expect(lexer.Eol) {
		return elseExpression, nil
	}

	elseControlFlow, err := p.ParseStatement()
	elseCompositeExpression, _ := elseControlFlow.(*GroupStatementNode)

	if elseCompositeExpression == nil {
		err = NewParseError(
			keywordToken,
			errors.New("else statement expect either 'if' or 'with' or nothing"),
		)
		err.Range.End = elseControlFlow.Range().End
		elseExpression.Err = err

		return elseExpression, err
	}

	// merge old token value with the newer one, separating them by a space ' '
	newValue := elseCompositeExpression.KeywordToken.Value
	elseCompositeExpression.KeywordToken = lexer.CloneToken(
		elseCompositeExpression.KeywordToken,
	)
	//nolint:gocritic // intentionally concatenating from elseExpression to elseCompositeExpression
	elseCompositeExpression.KeywordToken.Value = append(
		elseExpression.KeywordToken.Value,
		byte(' '),
	)
	elseCompositeExpression.KeywordToken.Value = append(
		elseCompositeExpression.KeywordToken.Value,
		newValue...)
	elseCompositeExpression.KeywordToken.Range.Start = elseExpression.rng.Start

	elseCompositeExpression.KeywordRange = elseCompositeExpression.KeywordToken.Range
	elseCompositeExpression.rng.Start = elseExpression.rng.Start
	elseCompositeExpression.Err = err

	switch elseCompositeExpression.Kind() {
	case KindIf:
		elseCompositeExpression.SetKind(KindElseIf)
	case KindWith:
		elseCompositeExpression.SetKind(KindElseWith)
	default:
		err = NewParseError(
			keywordToken,
			errors.New("else statement expect either 'if' or 'with' or nothing"),
		)
		err.Range = elseCompositeExpression.Range()
		elseCompositeExpression.Err = err
		return elseCompositeExpression, err
	}

	if err != nil {
		return elseCompositeExpression, err
	}

	if elseCompositeExpression.Range().End != p.lastToken.Range.End {
		panic(
			"ending location mismatch between 'else if/with/...' statement and its expression\n" + elseCompositeExpression.String(),
		)
	}

	return elseCompositeExpression, nil
}

// parseRangeKeyword handles the "range" keyword.
func (p *Parser) parseRangeKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	rangeExpression := NewGroupStatementNode(
		KindRangeLoop,
		keywordToken.Range,
		p.stream,
	)
	rangeExpression.rng.End = p.lastToken.Range.End
	rangeExpression.KeywordRange = keywordToken.Range
	rangeExpression.KeywordToken = keywordToken

	p.nextToken()

	expression, err := p.ParseStatement()
	rangeExpression.ControlFlow = expression
	rangeExpression.Err = err

	rangeExpression.rng.End = expression.Range().End

	if expression == nil {
		panic("unexpected <nil> AST return while parsing 'range expression'")
	}

	if err != nil {
		return rangeExpression, err
	}

	if err = validateControlFlowExpression(expression, "range"); err != nil {
		rangeExpression.Err = err
		return rangeExpression, err
	}

	return rangeExpression, nil
}

// parseWithKeyword handles the "with" keyword.
func (p *Parser) parseWithKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	withExpression := NewGroupStatementNode(
		KindWith,
		keywordToken.Range,
		p.stream,
	)
	withExpression.rng.End = p.lastToken.Range.End
	withExpression.KeywordRange = keywordToken.Range
	withExpression.KeywordToken = keywordToken

	p.nextToken() // skip 'with' token

	expression, err := p.ParseStatement()
	withExpression.ControlFlow = expression
	withExpression.Err = err

	withExpression.rng.End = expression.Range().End

	if expression == nil {
		panic("unexpected <nil> AST return while parsing 'with expression'")
	}

	if err != nil {
		return withExpression, err
	}

	if err = validateControlFlowExpression(expression, "with"); err != nil {
		withExpression.Err = err
		return withExpression, err
	}

	return withExpression, nil
}

// parseBlockKeyword handles the "block" keyword.
func (p *Parser) parseBlockKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	blockExpression := NewGroupStatementNode(
		KindBlockTemplate,
		keywordToken.Range,
		p.stream,
	)
	blockExpression.rng.End = p.lastToken.Range.End
	blockExpression.KeywordRange = keywordToken.Range
	blockExpression.KeywordToken = keywordToken

	p.nextToken() // skip 'block' token

	if !p.accept(lexer.StringLit) {
		err := NewParseError(
			p.peek(),
			errors.New("'block' expect a string next to it"),
		)
		blockExpression.Err = err
		return blockExpression, err
	}

	templateExpression := NewTemplateStatementNode(
		KindBlockTemplate,
		p.peek().Range,
	)
	templateExpression.TemplateName = p.peek()
	templateExpression.parent = blockExpression
	blockExpression.ControlFlow = templateExpression

	p.nextToken()

	expression, err := p.ParseStatement()
	templateExpression.Expression = expression
	templateExpression.rng.End = expression.Range().End

	blockExpression.Err = err
	blockExpression.rng.End = expression.Range().End
	if expression == nil {
		panic("unexpected <nil> AST return while parsing 'block expression'")
	}

	if err != nil {
		return blockExpression, err
	}

	if err = validateControlFlowExpression(expression, "block"); err != nil {
		blockExpression.Err = err
		return blockExpression, err
	}

	return blockExpression, nil
}

// parseDefineKeyword handles the "define" keyword.
func (p *Parser) parseDefineKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	defineExpression := NewGroupStatementNode(
		KindDefineTemplate,
		keywordToken.Range,
		p.stream,
	)
	defineExpression.rng.End = p.lastToken.Range.End
	defineExpression.KeywordRange = keywordToken.Range
	defineExpression.KeywordToken = keywordToken

	p.nextToken() // skip 'define' token

	if !p.accept(lexer.StringLit) {
		err := NewParseError(
			p.peek(),
			errors.New("'define' expect a string next to it"),
		)
		defineExpression.Err = err
		return defineExpression, err
	}

	templateExpression := NewTemplateStatementNode(
		KindDefineTemplate,
		p.peek().Range,
	)
	templateExpression.TemplateName = p.peek()
	templateExpression.parent = defineExpression
	defineExpression.ControlFlow = templateExpression

	p.nextToken()

	if !p.expect(lexer.Eol) {
		err := NewParseError(
			p.peek(),
			errors.New("'define' does not accept any expression"),
		)
		err.Range.End = p.lastToken.Range.End
		defineExpression.Err = err
		return defineExpression, err
	}

	return defineExpression, nil
}

// parseTemplateKeyword handles the "template" keyword.
func (p *Parser) parseTemplateKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	templateExpression := NewTemplateStatementNode(
		KindUseTemplate,
		keywordToken.Range,
	)
	templateExpression.rng.End = p.lastToken.Range.End
	templateExpression.KeywordRange = keywordToken.Range

	p.nextToken() // skip 'template' tokens

	if !p.accept(lexer.StringLit) {
		err := NewParseError(p.peek(), errors.New("missing template name"))
		templateExpression.Err = err
		return templateExpression, err
	}

	templateExpression.TemplateName = p.peek()
	p.nextToken()

	if p.accept(lexer.Eol) {
		return templateExpression, nil
	}

	expression, err := p.expressionStatementParser()
	templateExpression.Expression = expression
	templateExpression.Err = err

	templateExpression.rng.End = expression.Range().End

	if expression == nil {
		panic("unexpected <nil> AST return while parsing 'template expression'")
	}

	if err != nil {
		return templateExpression, err
	}

	if err = validateControlFlowExpression(expression, "template"); err != nil {
		templateExpression.Err = err
		return templateExpression, err
	}

	if !p.expect(lexer.Eol) {
		err := NewParseError(p.peek(), errors.New("early template end"))
		err.Range.End = p.lastToken.Range.End
		templateExpression.Err = err
		return templateExpression, err
	}

	return templateExpression, nil
}

// parseEndKeyword handles the "end" keyword.
func (p *Parser) parseEndKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	endExpression := NewGroupStatementNode(KindEnd, keywordToken.Range, p.stream)
	endExpression.rng.End = p.lastToken.Range.End
	endExpression.KeywordRange = keywordToken.Range
	endExpression.KeywordToken = keywordToken

	p.nextToken() // skip 'end' token

	if !p.expect(lexer.Eol) {
		err := NewParseError(
			keywordToken,
			errors.New("'end' is a standalone command"),
		)
		err.Range.End = p.lastToken.Range.End
		endExpression.Err = err
		return endExpression, err
	}

	return endExpression, nil
}

// parseBreakKeyword handles the "break" keyword.
func (p *Parser) parseBreakKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	breakCommand := NewSpecialCommandNode(
		KindBreak,
		keywordToken,
		keywordToken.Range,
	)

	p.nextToken() // skip token

	if !p.expect(lexer.Eol) {
		err := NewParseError(
			keywordToken,
			errors.New("'break' is a standalone command"),
		)
		err.Range.End = p.lastToken.Range.End
		breakCommand.Err = err
		return breakCommand, err
	}

	return breakCommand, nil
}

// parseContinueKeyword handles the "continue" keyword.
func (p *Parser) parseContinueKeyword(keywordToken *lexer.Token) (AstNode, *ParseError) {
	continueCommand := NewSpecialCommandNode(
		KindContinue,
		keywordToken,
		keywordToken.Range,
	)

	p.nextToken() // skip token

	if !p.expect(lexer.Eol) {
		err := NewParseError(
			keywordToken,
			errors.New("'continue' is a standalone command"),
		)
		err.Range.End = p.lastToken.Range.End
		continueCommand.Err = err
		return continueCommand, err
	}

	return continueCommand, nil
}
