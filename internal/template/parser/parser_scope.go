package parser

import (
	"errors"
	"log"

	"github.com/pacer/gozer/internal/template/lexer"
)

// groupMerger builds the AST by tracking open scopes and linking control flow statements.
type groupMerger struct {
	openedNodeStack []*GroupStatementNode // stack of currently open scopes
	topLinkedGroup  []*GroupStatementNode // tracks opener groups for linking with else/end
}

func newGroupMerger() *groupMerger {
	rootGroup := NewGroupStatementNode(KindGroupStatement, lexer.Range{}, nil)
	rootGroup.isRoot = true

	groupNodeStack := make([]*GroupStatementNode, 0, 1)
	groupNodeStack = append(groupNodeStack, rootGroup)

	merger := &groupMerger{
		openedNodeStack: groupNodeStack,
		topLinkedGroup:  nil,
	}

	return merger
}

//nolint:dupl // KindElseIf and KindElseWith cases are intentionally similar
func (p *groupMerger) safelyGroupStatement(node AstNode) *ParseError {
	if node == nil {
		log.Printf("cannot add <nil> AST to group")
		panic("cannot add <nil> AST to group")
	}

	if len(p.openedNodeStack) == 0 {
		panic(
			"no initial scope available to hold the statements. There must always exist at least one 'scope/group' at any moment",
		)
	}

	if !p.openedNodeStack[0].isRoot {
		panic(
			"root node hasn't been marked as such; only the root node can be marked as 'isRoot'",
		)
	}

	// this is only useful to keep track of the top most group, so that 'end' statement can link to it later
	if len(p.topLinkedGroup) != len(p.openedNodeStack)-1 {
		panic(
			"size of open node stack does not match the one only containing opener group (e.g., if, range, with, define)",
		)
	}

	var err *ParseError
	var ROOT_SCOPE = p.openedNodeStack[0]

	stackSize := len(p.openedNodeStack)
	currentScope := p.openedNodeStack[stackSize-1]

	newScope, isScope := node.(*GroupStatementNode)

	if !isScope {
		appendStatementToCurrentScope(currentScope, node)
		err = appendStatementToScopeShortcut(currentScope, node)

		switch node.Kind() {
		case KindContinue, KindBreak:
			loopControl, ok := node.(*SpecialCommandNode)
			if !ok {
				panic(
					"expected 'SpecialCommandNode' to safelyGroupStatement, but found something else :: " + node.String(),
				)
			}

			found := false

			for index := len(p.openedNodeStack) - 1; index >= 0; index-- { // reverse looping
				scope := p.openedNodeStack[index]

				if scope.kind == KindRangeLoop {
					loopControl.Target = scope
					found = true

					break
				}
			}

			if !found {
				err = NewParseError(
					&lexer.Token{},
					errors.New("missing 'range' loop ancestor"),
				)
				err.Range = node.Range()
			}
		}
	} else {
		if newScope.IsRoot() {
			log.Printf(
				"non-root node cannot be flaged as 'root'.\n culprit node = %#v\n",
				newScope,
			)
			panic(
				"only the root node, can be marked as 'isRoot', but found it on non-root node",
			)
		}

		newScope.parent = currentScope
		isStatementAlreadyAppendedToParentScope := false

		switch newScope.Kind() {
		case KindIf, KindWith, KindRangeLoop, KindBlockTemplate, KindDefineTemplate:
			err = appendStatementToScopeShortcut(currentScope, newScope)
			appendStatementToCurrentScope(currentScope, newScope)
			isStatementAlreadyAppendedToParentScope = true
			newScope.parent = currentScope

			p.openedNodeStack = append(p.openedNodeStack, newScope)
			p.topLinkedGroup = append(p.topLinkedGroup, newScope)

		case KindElseIf:
			if stackSize >= 2 {
				switch currentScope.Kind() {
				case KindIf,
					KindElseIf: // Remove the last element from the stack and switch it with 'KindElseIf' scope
					scopeToClose := currentScope
					scopeToClose.rng.End = newScope.Range().Start
					scopeToClose.NextLinkedSibling = newScope

					parentScope := p.openedNodeStack[stackSize-2]
					p.openedNodeStack = p.openedNodeStack[:stackSize-1]

					newScope.parent = parentScope
					isStatementAlreadyAppendedToParentScope = true
					appendStatementToCurrentScope(parentScope, newScope)

					p.openedNodeStack = append(p.openedNodeStack, newScope)

					size := len(p.topLinkedGroup)
					newScope.NextLinkedSibling = p.topLinkedGroup[size-1]

				default:
					err = &ParseError{
						Range: newScope.KeywordRange, // currentScope.Range(),
						Err: errors.New(
							"not compatible with " + currentScope.Kind().String(),
						),
					}
				}
			} else {
				err = &ParseError{Range: newScope.KeywordRange, // newScope.Range(),
					Err: errors.New(
						"extraneous statement '" + newScope.Kind().String() + "'",
					)}
			}
		case KindElseWith:
			if stackSize >= 2 {
				switch currentScope.Kind() {
				case KindWith, KindElseWith:
					scopeToClose := currentScope
					scopeToClose.rng.End = newScope.Range().Start
					scopeToClose.NextLinkedSibling = newScope

					// Remove the last element from the stack and switch it with 'KindElseWith' scope
					parentScope := p.openedNodeStack[stackSize-2]
					p.openedNodeStack = p.openedNodeStack[:stackSize-1] // fold current scope

					newScope.parent = parentScope
					isStatementAlreadyAppendedToParentScope = true
					appendStatementToCurrentScope(parentScope, newScope)

					p.openedNodeStack = append(p.openedNodeStack, newScope)

					size := len(p.topLinkedGroup)
					newScope.NextLinkedSibling = p.topLinkedGroup[size-1]

				default:
					err = &ParseError{
						Range: newScope.KeywordRange, // currentScope.Range(),
						Err: errors.New(
							"not compatible with " + currentScope.Kind().String(),
						),
					}
				}
			} else {
				err = &ParseError{Range: newScope.KeywordRange, // newScope.Range(),
					Err: errors.New(
						"extraneous statement '" + newScope.Kind().String() + "'",
					)}
			}
		case KindElse:
			if stackSize >= 2 {
				switch currentScope.Kind() {
				case KindIf, KindElseIf, KindWith, KindElseWith, KindRangeLoop:
					// Remove the last element from the stack and switch it with 'KindElse' scope

					scopeToClose := currentScope
					scopeToClose.rng.End = newScope.Range().Start
					scopeToClose.NextLinkedSibling = newScope

					parentScope := p.openedNodeStack[stackSize-2]
					p.openedNodeStack = p.openedNodeStack[:stackSize-1] // fold previous scope

					newScope.parent = parentScope
					isStatementAlreadyAppendedToParentScope = true
					appendStatementToCurrentScope(parentScope, newScope)

					p.openedNodeStack = append(p.openedNodeStack, newScope)

					size := len(p.topLinkedGroup)
					newScope.NextLinkedSibling = p.topLinkedGroup[size-1]

				default:
					err = &ParseError{Range: newScope.KeywordRange, // newScope.Range(),
						Err: errors.New(
							"not compatible with " + currentScope.Kind().String(),
						)}
				}
			} else {
				err = &ParseError{Range: newScope.KeywordRange, // newScope.Range(),
					Err: errors.New(
						"extraneous statement '" + newScope.Kind().String() + "'",
					)}
			}
		case KindEnd:
			if stackSize >= 2 {
				scopeToClose := currentScope
				scopeToClose.rng.End = newScope.Range().Start
				scopeToClose.NextLinkedSibling = newScope

				parentScope := p.openedNodeStack[stackSize-2]
				p.openedNodeStack = p.openedNodeStack[:stackSize-1] // fold/close current scope

				newScope.parent = parentScope
				isStatementAlreadyAppendedToParentScope = true
				appendStatementToCurrentScope(parentScope, newScope)

				size := len(p.topLinkedGroup)
				newScope.NextLinkedSibling = p.topLinkedGroup[size-1]

				p.topLinkedGroup = p.topLinkedGroup[:size-1]
			} else {
				err = &ParseError{
					Range: newScope.Range(),
					Err:   errors.New("extraneous 'end' statement"),
				}
			}
		default:
			log.Printf("unhandled scope type error\n scope = %#v\n", newScope)
			panic(
				"scope type '" + newScope.Kind().
					String() +
					"' is not yet handled for statement grouping\n" + newScope.String(),
			)
		}

		// only do this if for some reasons the statement hasn't been added to any existing scope
		if !isStatementAlreadyAppendedToParentScope {
			appendStatementToCurrentScope(currentScope, newScope)
		}
	}

	if len(p.openedNodeStack) == 0 {
		panic(
			"'openedNodeStack' cannot be empty ! you have inadvertly close the 'root scope'. You should not interact with it",
		)
	}

	if ROOT_SCOPE != p.openedNodeStack[0] {
		log.Printf("root scope change error. new Root = %#v\n", p.openedNodeStack[0])
		panic(
			"error, the root scope have been modified. The root scope should never change under any circumstance",
		)
	}

	return err
}
