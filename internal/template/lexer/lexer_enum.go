package lexer

// Kind identifies the type of a lexer token (e.g., keyword, variable, operator).
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
	Eol         // End of template block (not a literal newline)
	Eof         // End of file
	StaticGroup // A group of tokens that can be evaluated at compile time
	ExpandableGroup
	NotFound
	Unexpected
)
