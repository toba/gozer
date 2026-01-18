package parser

// -----------
// Parser Kind
// -----------

// Parser configuration constants
const (
	// maxRecursionDepth limits parser recursion to prevent stack overflow.
	maxRecursionDepth = 15

	// maxExpressionTokens limits tokens in a single expression to detect infinite loops.
	maxExpressionTokens = 100

	// maxVariablesPerDeclaration limits variables in a single declaration/assignment.
	maxVariablesPerDeclaration = 2
)

const (
	KindVariableDeclaration Kind = iota
	KindVariableAssignment
	KindExpression
	KindMultiExpression
	KindComment

	KindGroupStatement

	KindIf
	KindElseIf
	KindElse

	KindWith
	KindElseWith

	KindRangeLoop

	KindDefineTemplate
	KindBlockTemplate
	KindUseTemplate

	KindEnd
	KindContinue
	KindBreak
)
