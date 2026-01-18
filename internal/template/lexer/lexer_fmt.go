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
