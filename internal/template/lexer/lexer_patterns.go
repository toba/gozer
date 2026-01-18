package lexer

import (
	"log"
	"regexp"
)

// compiledPattern holds a pre-compiled regex pattern and its associated token information.
type compiledPattern struct {
	Regex                *regexp.Regexp
	ID                   Kind
	CanBeRightAfterToken bool
}

// compiledPatterns holds all pre-compiled regex patterns used during tokenization.
// These are initialized once at package load time to avoid repeated compilation.
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

	// Pre-compile token patterns
	// (tokenRecognizerPattern) Tokens' meaning: VariableName, ID (Function ?), '==' '=' ':='
	keywordPattern := "if|else|end|range|define|template|block|with|continue|break"
	compiledPatterns.tokenPatterns = []compiledPattern{
		{
			Regex: regexp.MustCompile(keywordPattern),
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
		// BUG: what if the user input multiple Character within delimitator ? A bug will appear
		// Solve it later
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
