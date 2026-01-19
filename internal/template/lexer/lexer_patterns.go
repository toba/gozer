package lexer

import (
	"log"
	"regexp"
)

// compiledPattern pairs a regex with metadata for matching a specific token type.
type compiledPattern struct {
	Regex                *regexp.Regexp
	ID                   Kind
	CanBeRightAfterToken bool
}

// compiledPatterns holds pre-compiled regex patterns, initialized once at package load.
var compiledPatterns struct {
	whitespace        *regexp.Regexp
	loneDelimiter     *regexp.Regexp
	templateStatement *regexp.Regexp
	tokenPatterns     []compiledPattern
}

func init() {
	// Pre-compile whitespace pattern
	compiledPatterns.whitespace = regexp.MustCompile(`\s+`)

	// Pre-compile template extraction patterns
	compiledPatterns.loneDelimiter = regexp.MustCompile("{{|}}")
	compiledPatterns.templateStatement = regexp.MustCompile(
		"(?:{{(?:[^{}]|[^{}]{|[^{}]}|[^{}]{}|[^{}]}{|[\n\r\t])*?}})",
	)

	// Token patterns (order matters: more specific patterns must come first)
	compiledPatterns.tokenPatterns = []compiledPattern{
		{
			Regex: regexp.MustCompile(KeywordPattern),
			ID:    Keyword,
		},
		{
			Regex: regexp.MustCompile(`"(?:[^"\n\\]|\\.)*"`),
			ID:    StringLit,
		},
		{
			Regex: regexp.MustCompile(`\x60(?:[^\x60\n\\]|\\.)*\x60`), // \x60 == \`
			ID:    StringLit,
		},
		{
			Regex: regexp.MustCompile(`'[^'\n\\]'`),
			ID:    Character,
		},
		{
			Regex: regexp.MustCompile(`(?:\d+|\d*[.]\d+)i`),
			ID:    ComplexNumber,
		},
		{
			Regex: regexp.MustCompile(`\d*[.]\d+`),
			ID:    Decimal,
		},
		{
			Regex: regexp.MustCompile(`\d+`),
			ID:    Number,
		},
		{
			Regex: regexp.MustCompile("true|false"),
			ID:    Boolean,
		},
		{
			Regex: regexp.MustCompile(`[$][.]?\w+(?:[.][a-zA-Z_]\w*)*|[$]`),
			ID:    DollarVariable,
		},
		{
			Regex: regexp.MustCompile(`(?:[.][a-zA-Z_]\w*)+|[.]`),
			ID:    DotVariable,
		},
		{
			Regex: regexp.MustCompile(`[[:alpha:]]\w*(?:[.][[:alpha:]]\w*)*`),
			ID:    Function,
		},
		{
			Regex:                regexp.MustCompile("=="),
			ID:                   EqualComparison,
			CanBeRightAfterToken: true,
		},
		{
			Regex:                regexp.MustCompile("="),
			ID:                   Assignment,
			CanBeRightAfterToken: true,
		},
		{
			Regex:                regexp.MustCompile(":="),
			ID:                   DeclarationAssignment,
			CanBeRightAfterToken: true,
		},
		{
			Regex:                regexp.MustCompile("[|]"),
			ID:                   Pipe,
			CanBeRightAfterToken: true,
		},
		{
			Regex:                regexp.MustCompile(`\(`),
			ID:                   LeftParen,
			CanBeRightAfterToken: true,
		},
		{
			Regex:                regexp.MustCompile(`\)`),
			ID:                   RightParen,
			CanBeRightAfterToken: true,
		},
		{
			Regex: regexp.MustCompile(`\/\*(?:.|\s)*?(?:\*\/)`),
			ID:    Comment,
		},
		{
			Regex: regexp.MustCompile(`,`),
			ID:    Comma,
		},
	}
}

// tokenizer accumulates tokens and errors while processing a single template block.
type tokenizer struct {
	Tokens     []Token
	Errs       []Error
	FirstError *LexerError
	LastToken  *Token
}

func (t *tokenizer) appendToken(id Kind, pos Range, val []byte) {
	to := Token{
		ID:    id,
		Range: pos,
		Value: val,
	}

	t.Tokens = append(t.Tokens, to)
	t.LastToken = &to
}

func (t *tokenizer) appendError(err error, token *Token) {
	if err == nil {
		log.Printf(
			"line tokenizer expected an error but got <nil> while appending error\n",
		)
		panic("line tokenizer expected an error but got <nil> while appending error")
	}

	lexErr := &LexerError{
		Err:   err,
		Token: token,
		Range: token.Range,
	}

	t.Errs = append(t.Errs, lexErr)

	if t.FirstError == nil {
		t.FirstError = lexErr
	}
}

func createTokenizer() *tokenizer {
	return &tokenizer{
		Tokens:     nil,
		Errs:       nil,
		FirstError: nil,
	}
}

// trimSuperflousCharacter strips delimiters from token values (e.g., quotes from strings).
func trimSuperflousCharacter(text []byte, id Kind) []byte {
	switch id {
	case Comment:
		lower := 2
		upper := len(text) - 2
		text = text[lower:upper]
	case StringLit:
		lower := 1
		upper := len(text) - 1
		text = text[lower:upper]
	}

	return text
}
