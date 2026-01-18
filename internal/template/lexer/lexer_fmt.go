package lexer

import (
	"fmt"
	"strings"
)

func (e LexerError) String() string {
	return fmt.Sprintf(
		`{ "Err": "%s", "Range": %s, "Token": %s }`,
		e.Err.Error(),
		e.Range,
		e.Token,
	)
}

func (p Position) String() string {
	return fmt.Sprintf("{ \"Line\": %d, \"Character\": %d }", p.Line, p.Character)
}

func (r Range) String() string {
	return fmt.Sprintf("{ \"Start\": %s, \"End\": %s }", r.Start, r.End)
}

func (t Token) String() string {
	return fmt.Sprintf(
		"{ \"ID\": \"%s\", \"Range\": %s, \"Value\": %q }",
		t.ID,
		t.Range,
		t.Value,
	)
}

func (k Kind) String() string {
	var str string

	switch k {
	case DotVariable:
		str = "DotVariable"
	case DollarVariable:
		str = "DollarVariable"
	case Keyword:
		str = "Keyword"
	case Function:
		str = "Function"
	case Identifier:
		str = "Identifier"
	case Assignment:
		str = "Assignment"
	case DeclarationAssignment:
		str = "DeclarationAssignment"
	case StringLit:
		str = "StringLit"
	case Number:
		str = "Number"
	case EqualComparison:
		str = "EqualComparison"
	case Pipe:
		str = "Pipe"
	case LeftParen:
		str = "LeftParen"
	case RightParen:
		str = "RightParen"
	case Comment:
		str = "Comment"
	case Eol:
		str = "Eol"
	case NotFound:
		str = "NotFound"
	case Unexpected:
		str = "Unexpected"
	case Comma:
		str = "Comma"
	case StaticGroup:
		str = "StaticGroup"
	case ExpandableGroup:
		str = "ExpandableGroup"
	case Character:
		str = "Character"
	case ComplexNumber:
		str = "ComplexNumber"
	case Decimal:
		str = "Decimal"
	case Boolean:
		str = "Boolean"
	case SpaceEater:
		str = "SpaceEater"
	case Eof:
		str = "Eof"
	default:
		str = fmt.Sprintf(
			"stringer() for 'lexer.Kind' type have found an unpected value: %d",
			k,
		)
		panic(str)
	}

	return str
}

// PrettyFormater converts an array of Stringer elements to a formatted string.
func PrettyFormater[T fmt.Stringer](arr []T) string {
	if len(arr) == 0 {
		return "[]"
	}

	str := "["
	var strSb84 strings.Builder
	for _, el := range arr {
		strSb84.WriteString(fmt.Sprintf("%s,", el))
	}
	str += strSb84.String()

	str = str[:len(str)-1]
	str += "]"

	return str
}

func Print(tokens ...Token) {
	str := PrettyFormater(tokens)
	fmt.Println(str)
}
