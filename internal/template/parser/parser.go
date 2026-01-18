package parser

import (
	"errors"
	"log"

	"github.com/pacer/gozer/internal/template/lexer"
)

var uniqueUniversalCounter int = 0

// GetUniqueNumber returns a unique integer at each call.
func GetUniqueNumber() int {
	uniqueUniversalCounter++
	return uniqueUniversalCounter
}

type ParseError struct {
	Err   error
	Range lexer.Range
	Token *lexer.Token
}

func (p ParseError) GetError() string {
	return p.Err.Error()
}

func (p ParseError) GetRange() lexer.Range {
	return p.Range
}

type Parser struct {
	stream            *lexer.StreamToken
	lastToken         *lexer.Token // token before 'EOL', and only computed once at 'Reset()'
	indexCurrentToken int
	sizeStream        int

	maxRecursionDepth     int
	currentRecursionDepth int
}

func (p *Parser) Reset(streamOfToken *lexer.StreamToken) {
	p.lastToken = nil
	p.stream = streamOfToken
	p.sizeStream = len(streamOfToken.Tokens)
	p.indexCurrentToken = 0

	if p.sizeStream >= 2 {
		p.lastToken = &p.stream.Tokens[p.sizeStream-2] // careful, last token is 'EOL'
	}

	if p.sizeStream < 1 { // even an empty statement have 'EOL'
		panic(
			"every token stream must at least have 'EOL' token, even an empty statement",
		)
	}

	p.maxRecursionDepth = maxRecursionDepth
	p.currentRecursionDepth = 0
}

func appendParseError(errs []lexer.Error, err *ParseError) []lexer.Error {
	if err == nil {
		return errs
	}

	return append(errs, err)
}

func appendStatementToScopeShortcut(
	scope *GroupStatementNode,
	statement AstNode,
) *ParseError {
	switch stmt := statement.(type) {
	case *GroupStatementNode:
		if !stmt.IsTemplate() {
			return nil
		}

		if stmt.ControlFlow == nil && stmt.Err != nil {
			return nil
		} else if stmt.ControlFlow == nil {
			panic(
				"expected non <nil> ControlFlow for 'GroupStatementNode' while appending it to parent scope",
			)
		}

		templateNode := stmt.ControlFlow.(*TemplateStatementNode)
		if templateNode == nil || templateNode.TemplateName == nil {
			return nil
		}

		templateName := string(templateNode.TemplateName.Value)

		if scope.ShortCut.TemplateDefined[templateName] != nil {
			err := NewParseError(
				templateNode.TemplateName,
				errors.New("template already defined"),
			)
			return err
		}

		scope.ShortCut.TemplateDefined[templateName] = stmt

	case *TemplateStatementNode:
		if stmt.TemplateName == nil {
			return nil
		}

		// Look for template group that will hold the template call shortcut
		// This assume the root group is always available for the program to not crash
		for !IsGroupNode(scope.kind) {
			scope = scope.parent
		}

		scope.ShortCut.TemplateCallUsed = append(scope.ShortCut.TemplateCallUsed, stmt)

	case *CommentNode:
		if stmt.GoCode == nil {
			return nil
		}

		if scope.ShortCut.CommentGoCode != nil {
			// stmt.GoCode = nil
			err := errors.New("cannot redeclare 'go:code' in the same scope")
			return NewParseError(stmt.Value, err)
		}

		scope.ShortCut.CommentGoCode = stmt

	case *VariableDeclarationNode:
		for _, variableToken := range stmt.VariableNames {
			variableName := string(variableToken.Value)

			scope.ShortCut.VariableDeclarations[variableName] = stmt
		}
	}

	return nil
}

func appendStatementToCurrentScope(scope *GroupStatementNode, statement AstNode) {
	if statement == nil {
		panic("cannot add empty statement to 'group'")
	}

	if scope == nil {
		panic("cannot add statement to empty group. Group must be created before hand")
	}

	scope.Statements = append(scope.Statements, statement)
	scope.rng.End = statement.Range().End
}

// Parse tokens into AST and return syntax errors found during the process
// Returned parse tree is never <nil>
func Parse(streams []*lexer.StreamToken) (*GroupStatementNode, []lexer.Error) {
	var errs []lexer.Error

	merger := newGroupMerger()
	parser := Parser{}

	if len(streams) == 0 {
		return merger.openedNodeStack[0], nil
	}

	// main processing
	for _, stream := range streams {
		parser.Reset(stream)

		node, err := parser.ParseStatement()
		node.SetError(err)

		if stream.Err == nil { // otherwise do not report the error since it was already done in lexing
			errs = appendParseError(errs, err)
		}

		if node == nil {
			log.Printf(
				"unexpected <nil> AST. Even partial ASP must be non <nil> for better code analysis\n statementToken = %#v\n fileToken = %#v\n",
				stream,
				streams,
			)
			panic(
				"unexpected <nil> AST. Even partial ASP must be non <nil> for better code analysis",
			)
		}

		err = merger.safelyGroupStatement(node)
		errs = appendParseError(errs, err)
	}

	if len(merger.openedNodeStack) == 0 {
		log.Printf(
			"fatal error while building the parse tree. Expected at least one scope/group but found nothing\n stream = %#v",
			streams,
		)
		panic(
			"fatal error while building the parse tree. Expected at least one scope/group but found nothing",
		)
	}

	var unclosedScopes []*GroupStatementNode = nil
	if size := len(merger.openedNodeStack); size > 1 {
		unclosedScopes = merger.openedNodeStack[1:]
	}

	// for _, currentScope := range unclosedScopes {
	for index := len(unclosedScopes) - 1; index >= 0; index-- {
		// very imporant to start from the back
		// since the last element have the most accurate end of range value
		currentScope := unclosedScopes[index]

		size := len(currentScope.Statements)
		if size > 0 {
			lastStatement := currentScope.Statements[size-1]
			currentScope.rng.End = lastStatement.Range().End
		}

		err := NewParseError(
			currentScope.KeywordToken,
			errors.New("missing matching '{{ end }}' statement"),
		)
		errs = append(errs, err)
	}

	defaultGroupStatementNode := merger.openedNodeStack[0]
	if size := len(defaultGroupStatementNode.Statements); size > 0 {
		defaultGroupStatementNode.rng.Start = defaultGroupStatementNode.Statements[0].Range().Start
		defaultGroupStatementNode.rng.End = defaultGroupStatementNode.Statements[size-1].Range().End
	}

	return defaultGroupStatementNode, errs
}
