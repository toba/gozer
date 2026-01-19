package analyzer

import (
	"bytes"
	"fmt"
	"go/types"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

// InferenceFunc attempts to infer a symbol's type from a constraint.
type InferenceFunc func(symbol *lexer.Token, symbolType, constraintType types.Type) (*collectionPostCheckImplicitTypeNode, *parser.ParseError)

// InferenceFoundReturn holds type inference results from analyzing an expression.
type InferenceFoundReturn struct {
	uniqueVariableInExpression     *collectionPostCheckImplicitTypeNode   // primary variable for type propagation
	variablesToRecheckAtEndOfScope []*collectionPostCheckImplicitTypeNode // deferred type checks resolved at scope end
}

// OperatorType identifies how to compare types during deferred type checking.
type OperatorType int

// collectionPostCheckImplicitTypeNode stores a deferred type check between a candidate
// and constraint type, resolved at the end of the enclosing scope.
type collectionPostCheckImplicitTypeNode struct {
	candidate       *nodeImplicitType // Warning, this implicit node is not related to its variable definition
	candidateDef    *VariableDefinition
	candidateSymbol *lexer.Token

	constraint       *nodeImplicitType
	constraintDef    *VariableDefinition
	constraintSymbol *lexer.Token

	operation        OperatorType
	isAssignmentNode bool
}

func newCollectionPostCheckImplicitTypeNode(
	candidateNode, constraintNode *nodeImplicitType,
	candidateDef, constraintDef *VariableDefinition,
	candidateToken, constraintToken *lexer.Token,
) *collectionPostCheckImplicitTypeNode {
	if candidateToken == nil {
		panic("candidate token to recheck later is <nil>")
	} else if candidateNode == nil {
		panic("candidate implicit tree type to recheck later is <nil>")
	} else if candidateDef == nil {
		panic("candidate variable definition to recheck later is <nil>")
	} else if constraintNode == nil {
		panic("constraint implicit tree type to recheck later is <nil>")
	}

	ephemeral := extractOrInsertTemporaryImplicitTypeFromVariable(
		candidateDef,
		candidateToken,
	)
	if candidateNode != ephemeral {
		panic(
			"node tree extracted from 'candidateDef' does not correspond to the received one",
		)
	}

	collection := &collectionPostCheckImplicitTypeNode{
		candidate:       candidateNode,
		candidateDef:    candidateDef,
		candidateSymbol: candidateToken,

		constraint:       constraintNode,
		constraintDef:    constraintDef,
		constraintSymbol: constraintToken,

		operation:        operatorStrictType,
		isAssignmentNode: false,
	}

	return collection
}

// NodeDefinition is implemented by all symbol definitions (variables, functions, templates).
type NodeDefinition interface {
	Name() string
	FileName() string
	Node() parser.AstNode
	Range() lexer.Range
	Type() types.Type
	TypeString() string
}

// BasicSymbolDefinition represents a literal value (string, int, bool, etc.).
type BasicSymbolDefinition struct {
	node     parser.AstNode
	rng      lexer.Range
	fileName string
	name     string
	typ      *types.Basic
}

func (b *BasicSymbolDefinition) Name() string {
	return b.name
}

func (b *BasicSymbolDefinition) FileName() string {
	return b.fileName
}

func (b *BasicSymbolDefinition) Type() types.Type {
	return b.typ
}

func (b *BasicSymbolDefinition) Node() parser.AstNode {
	return b.node
}

func (b *BasicSymbolDefinition) Range() lexer.Range {
	return b.rng
}

func (b *BasicSymbolDefinition) TypeString() string {
	return fmt.Sprintf("var _ %s = %s", b.typ.String(), b.name)
}

// KeywordSymbolDefinition represents a template keyword (if, range, end, etc.).
type KeywordSymbolDefinition struct {
	node     parser.AstNode
	rng      lexer.Range
	fileName string
	name     string
}

func (k *KeywordSymbolDefinition) Name() string {
	return k.name
}

func (k *KeywordSymbolDefinition) FileName() string {
	return k.fileName
}

func (k *KeywordSymbolDefinition) Type() types.Type {
	return types.Typ[types.Invalid]
}

func (k *KeywordSymbolDefinition) Node() parser.AstNode {
	return k.node
}

func (k *KeywordSymbolDefinition) Range() lexer.Range {
	return k.rng
}

func (k *KeywordSymbolDefinition) TypeString() string {
	return k.name
}

func NewKeywordSymbolDefinition(
	name string,
	fileName string,
	node parser.AstNode,
) *KeywordSymbolDefinition {
	def := &KeywordSymbolDefinition{
		name:     name,
		node:     node,
		rng:      node.Range(),
		fileName: fileName,
	}

	return def
}

// FunctionDefinition represents a template function (built-in or custom).
type FunctionDefinition struct {
	node     parser.AstNode
	rng      lexer.Range
	fileName string
	name     string
	typ      *types.Signature
}

func (f *FunctionDefinition) Name() string {
	return f.name
}

func (f *FunctionDefinition) FileName() string {
	return f.fileName
}

func (f *FunctionDefinition) Type() types.Type {
	return f.typ
}

func (f *FunctionDefinition) Node() parser.AstNode {
	return f.node
}

func (f *FunctionDefinition) Range() lexer.Range {
	return f.rng
}

func (f *FunctionDefinition) TypeString() string {
	buf := new(bytes.Buffer)
	types.WriteSignature(buf, f.typ, nil)

	str := "func " + f.name + " " + buf.String()

	return str
}

// VariableDefinition represents a declared type (type known at declaration)
// or inferred type (type deduced by compiler).
type VariableDefinition struct {
	node       parser.AstNode // direct node containing info about this variable
	parent     *parser.GroupStatementNode
	rng        lexer.Range // variable lifetime
	fileName   string
	name       string
	typ        types.Type // declared type
	shadowType types.Type

	TreeImplicitType *nodeImplicitType // inferred type
	IsUsedOnce       bool              // Only useful to detect whether or not a variable have never been used in the scope
}

func (v *VariableDefinition) Name() string {
	return v.name
}

func (v *VariableDefinition) FileName() string {
	return v.fileName
}

func (v *VariableDefinition) Type() types.Type {
	return v.typ
}

func (v *VariableDefinition) Node() parser.AstNode {
	return v.node
}

func (v *VariableDefinition) Parent() *parser.GroupStatementNode {
	return v.parent
}

func (v *VariableDefinition) Range() lexer.Range {
	return v.rng
}

func (v *VariableDefinition) TypeString() string {
	str := v.typ.String()

	switch v.typ.(type) {
	case *types.Named:
		str = v.typ.Underlying().String()

	default:
	}

	str = "var " + v.name + " " + str

	return str
}

// TemplateDefinition represents a defined template ({{define "name"}} or {{block "name"}}).
type TemplateDefinition struct {
	node      parser.AstNode
	rng       lexer.Range
	fileName  string
	name      string
	inputType types.Type
}

func (t *TemplateDefinition) Name() string {
	return t.name
}

func (t *TemplateDefinition) FileName() string {
	return t.fileName
}

func (t *TemplateDefinition) Type() types.Type {
	return t.inputType
}

func (t *TemplateDefinition) Node() parser.AstNode {
	return t.node
}

func (t *TemplateDefinition) Range() lexer.Range {
	return t.rng
}

// TypeString returns a string representation of the template's type.
func (t *TemplateDefinition) TypeString() string {
	str := "var _ " + t.inputType.Underlying().String()

	return str
}

// NewTemplateDefinition creates a new TemplateDefinition with the given parameters.
func NewTemplateDefinition(
	name string,
	fileName string,
	node parser.AstNode,
	rng lexer.Range,
	typ types.Type,
) *TemplateDefinition {
	def := &TemplateDefinition{}
	def.name = name
	def.fileName = fileName
	def.node = node
	def.rng = rng
	def.inputType = typ

	return def
}

// FileDefinition holds analysis results for a single template file.
type FileDefinition struct {
	root                                       *parser.GroupStatementNode
	name                                       string
	typeHints                                  map[*parser.GroupStatementNode]types.Type // inferred dot type per scope
	scopeToVariables                           map[*parser.GroupStatementNode]map[string]*VariableDefinition
	functions                                  map[string]*FunctionDefinition
	templates                                  map[string]*TemplateDefinition
	isTemplateGroupAlreadyAnalyzed             bool
	extraVariableNameWithTypeInferenceBehavior map[string]*VariableDefinition // only useful to allow type inference on 'key' of the loop
	secondaryVariable                          *VariableDefinition            // only useful for passing around the 'key' of the loop
	// WorkspaceTemplates	map[string]*TemplateDefinition
}

func (f *FileDefinition) Name() string {
	return f.name
}

func (f *FileDefinition) FileName() string {
	return f.name
}

func (f *FileDefinition) Type() types.Type {
	return f.typeHints[f.root]
}

func (f *FileDefinition) Node() parser.AstNode {
	return f.root
}

func (f *FileDefinition) Range() lexer.Range {
	return f.root.Range()
}

func (f *FileDefinition) TypeString() string {
	return f.Type().String()
}

func (f *FileDefinition) Root() *parser.GroupStatementNode {
	return f.root
}

// GetScopedVariables returns variables declared directly in the given scope.
func (f *FileDefinition) GetScopedVariables(
	scope *parser.GroupStatementNode,
) map[string]*VariableDefinition {
	scopedVariables := f.scopeToVariables[scope]
	if scopedVariables == nil {
		scopedVariables = make(map[string]*VariableDefinition)
	}

	return scopedVariables
}

func (f *FileDefinition) GetVariableDefinitionWithinScope(
	variableName string,
	scope *parser.GroupStatementNode,
) *VariableDefinition {
	var count = 0

	for scope != nil {
		count++
		if count > MaxLoopRepetition {
			panic(
				"possible infinite loop detected while processing 'GetVariableDefinitionWithinScope()'",
			)
		}

		scopedVariables := f.GetScopedVariables(scope)

		varDef, ok := scopedVariables[variableName]
		if ok {
			return varDef
		}

		if scope.IsTemplate() {
			break
		}

		scope = scope.Parent()
	}

	return nil
}
