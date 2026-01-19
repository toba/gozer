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

// PrettyFormatter formats a slice of Stringer values as a comma-separated list in brackets.
func PrettyFormatter[T fmt.Stringer](arr []T) string {
	if len(arr) == 0 {
		return "[]"
	}

	str := "["
	var strSb84 strings.Builder
	for _, el := range arr {
		fmt.Fprintf(&strSb84, "%s,", el)
	}
	str += strSb84.String()

	str = str[:len(str)-1]
	str += "]"

	return str
}

// Print outputs tokens as a formatted list to stdout (for debugging).
func Print(tokens ...Token) {
	str := PrettyFormatter(tokens)
	fmt.Println(str)
}
