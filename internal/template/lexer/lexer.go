package lexer

import (
	"bytes"
	"log"
)

// ----------------------
// Lexer Types definition
// ----------------------

type Position struct {
	Line      int
	Character int
}

type Range struct {
	Start Position
	End   Position
}

func (r Range) Contains(pos Position) bool {
	if r.Start.Line > pos.Line {
		return false
	}

	if r.End.Line < pos.Line {
		return false
	}

	if r.Start.Line == pos.Line && pos.Character < r.Start.Character {
		return false
	}

	if r.End.Line == pos.Line && pos.Character >= r.End.Character {
		return false
	}

	return true
}

func (r Range) IsEmpty() bool {
	return r.Start.Line == 0 && r.Start.Character == 0 && r.End.Line == 0 &&
		r.End.Character == 0
}

func EmptyRange() Range {
	return Range{}
}

// Offset returns a new Position with the character offset by delta.
func (p Position) Offset(delta int) Position {
	return Position{
		Line:      p.Line,
		Character: p.Character + delta,
	}
}

// AdjustStart returns a new Range with the start character offset by delta.
func (r Range) AdjustStart(delta int) Range {
	return Range{
		Start: r.Start.Offset(delta),
		End:   r.End,
	}
}

// AdjustEnd returns a new Range with the end character offset by delta.
func (r Range) AdjustEnd(delta int) Range {
	return Range{
		Start: r.Start,
		End:   r.End.Offset(delta),
	}
}

// Shrink returns a new Range with start advanced by startChars and end reduced by endChars.
func (r Range) Shrink(startChars, endChars int) Range {
	return Range{
		Start: r.Start.Offset(startChars),
		End:   r.End.Offset(-endChars),
	}
}

//go:generate stringer -type=Kind
type Kind int

type StreamToken struct {
	Tokens                  []Token
	Err                     *LexerError
	rng                     Range
	IndexFirstEqualOperator int
}

func (s StreamToken) IsEmpty() bool {
	if len(s.Tokens) == 0 {
		panic("token stream must at least have an 'EOL' token")
	}

	if len(s.Tokens) == 1 && s.Tokens[0].ID == Eol {
		return true
	}

	return false
}

func (s StreamToken) String() string {
	size := len(s.Tokens)

	if size == 0 {
		panic("token stream must at least have an 'EOL' token")
	} else if token := s.Tokens[size-1]; token.ID != Eol {
		panic("token stream must be terminated by an 'EOL' token")
	}

	str := ""
	for index := range size - 1 { // ignore last #EOL token
		tok := s.Tokens[index]

		piece := string(tok.Value)
		if tok.ID == StringLit {
			// piece = `"` + piece + `"`
			// piece = "\"" + piece + "\""
			// piece = "\\\"" + piece + "\\\""
			piece = "`" + piece + "`"
		}

		str = str + " " + piece
	}

	return str[1:]
}

type Token struct {
	ID    Kind
	Range Range
	Value []byte
}

func NewToken(id Kind, reach Range, val []byte) *Token {
	fresh := &Token{
		ID:    id,
		Range: reach,
		Value: val,
	}

	return fresh
}

func CloneToken(old *Token) *Token {
	if old == nil {
		return nil
	}

	fresh := &Token{
		ID:    old.ID,
		Range: old.Range,
		Value: bytes.Clone(old.Value),
	}

	return fresh
}

func NewStreamToken(tokens []Token, err *LexerError, reach Range, loc int) *StreamToken {
	stream := &StreamToken{
		Tokens:                  tokens,
		Err:                     err,
		rng:                     reach,
		IndexFirstEqualOperator: loc,
	}

	return stream
}

type LexerError struct {
	Err   error
	Range Range
	Token *Token
}

func (l LexerError) GetError() string {
	return l.Err.Error()
}

func (l LexerError) GetRange() Range {
	return l.Range
}

type Error interface {
	GetError() string
	GetRange() Range
	String() string
}

// Tokenize the source code provided by 'content'.
// Each template pair delimitator ('{{' and '}}') represent an instruction of statement.
// Each source code instruction is tokenized separately, and the output are tokens representing the instruction.
// Every tokens representing an instruction always end by a 'EOL' tokens
// To sum up, the lexer/tokenizer return an array of token stream representing all instruction inside a file
func Tokenize(content []byte) (file []*StreamToken, errs []Error) {
	if len(content) == 0 {
		return nil, nil
	}

	templateCodes, templatePositions := extractTemplateCode(content)

	if templateCodes == nil {
		return nil, nil
	}

	var lineEndToken Token

	for i := range templateCodes {
		code := templateCodes[i]
		position := templatePositions[i]

		stream, tokenErrs := tokenizeLine(code, position)

		if stream == nil {
			log.Printf(
				"Unexpected <nil> token stream found at end of tokenizer process\n line = %q\n fileContent = %q\n",
				code,
				content,
			)
			panic("Unexpected <nil> token stream found at end of tokenizer process")
		}

		lineEndToken = Token{ID: Eol, Value: []byte("#EOL"), Range: position}
		stream.Tokens = append(stream.Tokens, lineEndToken)

		errs = append(errs, tokenErrs...)
		file = append(file, stream)
	}

	return file, errs
}
