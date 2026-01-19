package parser

import (
	"fmt"
	"strings"

	"github.com/pacer/gozer/internal/template/lexer"
)

func (e ParseError) String() string {
	to := "\"\""
	err := "\"\""

	if e.Err != nil {
		err = e.Err.Error()
		err = strings.ReplaceAll(err, "\"", "'")
	}
	if e.Token != nil {
		to = fmt.Sprint(*e.Token)
	}

	return fmt.Sprintf(`{"Err": "%s", "Range": %s, "Token": %s}`, err, e.Range, to)
}

func (v VariableDeclarationNode) String() string {
	value := `""`
	if v.Value != nil {
		value = fmt.Sprint(v.Value)
	}
	str := lexer.PrettyFormatter(v.VariableNames)

	return fmt.Sprintf(
		`{"Kind": %s, "Range": %s, "VariableNames": %s, "Value": %s}`,
		v.kind,
		v.rng,
		str,
		value,
	)
}

func (v VariableAssignationNode) String() string {
	variableName := lexer.PrettyFormatter(v.VariableNames)

	value := `""`
	if v.Value != nil {
		value = fmt.Sprint(v.Value)
	}

	return fmt.Sprintf(
		`{"Kind": %s, "Range": %s, "VariableName": %s, "Value": %s}`,
		v.kind,
		v.rng,
		variableName,
		value,
	)
}

func (m MultiExpressionNode) String() string {
	str := ""

	if len(m.Expressions) == 0 {
		str = "[]"
	} else {
		var strSb53 strings.Builder
		for _, expression := range m.Expressions {
			fmt.Fprintf(&strSb53, "%s, ", expression)
		}
		str += strSb53.String()

		str = "[" + str[:len(str)-2] + "]"
	}

	return fmt.Sprintf(`{"Kind": %s, "Range": %s, "Expressions": %s}`, m.kind, m.rng, str)
}

func (e ExpressionNode) String() string {
	str := ""

	if len(e.Symbols) == 0 {
		str = "[]"
	} else {
		var strSb69 strings.Builder
		for _, symbol := range e.Symbols {
			fmt.Fprintf(&strSb69, "%s, ", symbol)
		}
		str += strSb69.String()

		str = "[" + str[:len(str)-2] + "]"
	}

	return fmt.Sprintf(`{ "Kind": %s, "Range": %s, "Symbols": %s }`, e.kind, e.rng, str)
}

func (t TemplateStatementNode) String() string {
	templateName := `""`
	expression := `""`
	if t.TemplateName != nil {
		templateName = fmt.Sprint(t.TemplateName)
	}
	if t.Expression != nil {
		expression = fmt.Sprint(t.Expression)
	}

	return fmt.Sprintf(
		`{"Kind": %s, "Range": %s, "templateName": %s, "expression": %s}`,
		t.kind,
		t.rng,
		templateName,
		expression,
	)
}

func (g GroupStatementNode) String() string {
	strControlFlow := "{}"
	if g.ControlFlow != nil {
		strControlFlow = g.ControlFlow.String()
	}
	str := PrettyAstNodeFormater(g.Statements)

	return fmt.Sprintf(
		`{"Kind": %s, "Range": %s, "controlFlow": %s, "Statements": %s}`,
		g.kind,
		g.rng,
		strControlFlow,
		str,
	)
}

func (c CommentNode) String() string {
	value := `""`
	if c.Value != nil {
		value = fmt.Sprint(*c.Value)
	}

	return fmt.Sprintf(`{"Kind": %s, "Range": %s, "Value": %s}`, c.kind, c.rng, value)
}

func (s SpecialCommandNode) String() string {
	return fmt.Sprintf(
		`{"Kind": %s, "Range": %s, "Value": %s, "Err": "%s"}`,
		s.kind,
		s.rng,
		s.Value,
		s.Err,
	)
}

func PrettyAstNodeFormater(nodes []AstNode) string {
	str := ""

	if len(nodes) == 0 {
		str = "[]"
	} else {
		var strSb122 strings.Builder
		for _, node := range nodes {
			fmt.Fprintf(&strSb122, "%s, ", node)
		}
		str += strSb122.String()

		str = "[" + str[:len(str)-2] + "]"
	}

	return str
}

func PrettyFormatter[E fmt.Stringer](nodes []E) string {
	str := ""

	if len(nodes) == 0 {
		str = "[]"
	} else {
		var strSb138 strings.Builder
		for _, node := range nodes {
			fmt.Fprintf(&strSb138, "%v, ", node)
			// str += fmt.Sprintf("%#v, ", node)
		}
		str += strSb138.String()

		str = "[" + str[:len(str)-2] + "]"
	}

	return str
}

func Print(nodes ...AstNode) {
	str := PrettyAstNodeFormater(nodes)
	fmt.Println(str)
}

func (k Kind) String() string {
	val := "NOT FOUND!!!!!!!"

	switch k {
	case KindExpression:
		val = "KindExpression"
	case KindMultiExpression:
		val = "KindMultiExpression"
	case KindVariableAssignment:
		val = "KindVariableAssignment"
	case KindVariableDeclaration:
		val = "KindVariableDeclaration"
	case KindGroupStatement:
		val = "KindGroupStatement"
	case KindComment:
		val = "KindComment"
	case KindIf:
		val = "KindIf"
	case KindElseIf:
		val = "KindElseIf"
	case KindElse:
		val = "KindElse"
	case KindWith:
		val = "KindWith"
	case KindElseWith:
		val = "KindElseWith"
	case KindBlockTemplate:
		val = "KindBlockTemplate"
	case KindRangeLoop:
		val = "KindRangeLoop"
	case KindDefineTemplate:
		val = "KindDefineTemplate"
	case KindUseTemplate:
		val = "KindUseTemplate"
	case KindEnd:
		val = "KindEnd"
	case KindContinue:
		val = "KindContinue"
	case KindBreak:
		val = "KindBreak"
	}

	return fmt.Sprintf(`"%s"`, val)
}
