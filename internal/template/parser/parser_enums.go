package parser

// -----------
// Parser Kind
// -----------

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
