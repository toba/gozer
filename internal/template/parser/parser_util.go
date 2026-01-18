package parser

import (
	"bytes"
	"errors"

	"github.com/pacer/gozer/internal/template/lexer"
)

func lookForAndSetGoCodeInComment(commentExpression *CommentNode) {
	const SEP_COMMENT_GOCODE = "go:code"
	comment := commentExpression.Value.Value

	before, after, found := bytes.Cut(comment, []byte(SEP_COMMENT_GOCODE))
	if !found {
		return
	} else if len(after) == 0 {
		return
	}

	if len(bytes.TrimSpace(before)) > 0 {
		return
	}

	// if bytes.TrimSpace(before) == 0; then execute below
	comment = after

	switch comment[0] {
	case ' ', '\n', '\t', '\r', '\v', '\f': // unicode.IsSpace(rune(comment[0]))
		// continue to next step successfully
	default:
		return
	}

	initialLength := len(commentExpression.Value.Value)
	finalLength := len(comment)
	indexStartGoCode := initialLength - finalLength

	relativePositionStartGoCode := lexer.ConvertSingleIndexToTextEditorPosition(
		commentExpression.Value.Value,
		indexStartGoCode,
	)

	reach := commentExpression.rng
	reach.Start.Line += relativePositionStartGoCode.Line
	reach.Start.Character = relativePositionStartGoCode.Character

	commentExpression.GoCode = &lexer.Token{
		ID:    lexer.Comment,
		Range: reach,
		Value: comment,
	}
}

func (p Parser) peek() *lexer.Token {
	index := p.indexCurrentToken

	if index >= p.sizeStream {
		return nil
	}

	return &p.stream.Tokens[index]
}

func (p *Parser) nextToken() {
	p.indexCurrentToken++
}

func (p Parser) accept(kind lexer.Kind) bool {
	index := p.indexCurrentToken

	if index >= p.sizeStream {
		return false
	}

	return p.stream.Tokens[index].ID == kind
}

func (p *Parser) expect(kind lexer.Kind) bool {
	if p.accept(kind) {
		p.nextToken()

		return true
	}

	return false
}

func (p *Parser) incRecursionDepth() {
	p.currentRecursionDepth++
}

func (p Parser) checkRecursionStatus() (*MultiExpressionNode, *ParseError) {
	if p.isRecursionMaxDepth() {
		err := NewParseError(
			p.peek(),
			errors.New("parser error, reached the max depth authorized"),
		)
		multiExpression := NewMultiExpressionNode(
			KindMultiExpression,
			p.peek().Range.Start,
			p.lastToken.Range.End,
			err,
		)
		return multiExpression, err
	}

	return nil, nil
}

func (p *Parser) decRecursionDepth() {
	p.currentRecursionDepth--
}

func (p Parser) isRecursionMaxDepth() bool {
	return p.currentRecursionDepth >= p.maxRecursionDepth
}

func NewParseError(token *lexer.Token, err error) *ParseError {
	if token == nil {
		panic("token cannot be nil while creating parse error")
	}

	e := &ParseError{
		Err:   err,
		Range: token.Range,
		Token: token,
	}

	return e
}

// validControlFlowKinds defines which AST node kinds are valid in control flow expressions.
var validControlFlowKinds = map[Kind]bool{
	KindVariableAssignment:  true,
	KindVariableDeclaration: true,
	KindMultiExpression:     true,
	KindExpression:          true,
}

// validateControlFlowExpression checks if an expression is valid for control flow statements.
// Returns a ParseError if the expression kind is not allowed.
func validateControlFlowExpression(expression AstNode, keyword string) *ParseError {
	if validControlFlowKinds[expression.Kind()] {
		return nil
	}

	err := NewParseError(
		&lexer.Token{},
		errors.New("'"+keyword+"' does not accept this type of expression"),
	)
	err.Range = expression.Range()

	return err
}
