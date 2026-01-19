package analyzer

// Analysis limits to prevent infinite loops and excessive recursion.
const (
	// MaxProjectFileDepth limits directory recursion when scanning project files.
	MaxProjectFileDepth = 5

	// MaxDependencyDepth limits template dependency analysis recursion.
	MaxDependencyDepth = 20

	// MaxLoopRepetition limits iterations when walking parent scopes.
	MaxLoopRepetition = 20
)
