package lexer

// ----------
// Lexer Kind
// ----------

const (
	DotVariable Kind = iota
	DollarVariable
	Keyword
	Function
	Identifier
	Assignment
	DeclarationAssignment
	StringLit
	Character
	Number
	Decimal
	ComplexNumber
	Boolean
	EqualComparison
	Pipe
	Comma
	LeftParen
	RightParen
	Comment
	SpaceEater
	Eol // End Of Line
	Eof
	// StaticGroup represents a static group.
	StaticGroup
	ExpandableGroup
	NotFound
	Unexpected
)
