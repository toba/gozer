package analyzer

import (
	"go/token"
	"go/types"
	"log"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

var (
	typeAny                                   = types.Universe.Lookup("any")
	typeError                                 = types.Universe.Lookup("error")
	TemplateManager *WorkspaceTemplateManager = nil

	// customFunctions holds custom template functions discovered by scanning Go source files.
	// These are merged with builtin functions during analysis.
	customFunctions map[string]*FunctionDefinition
)

const (
	operatorStrictType OperatorType = iota
	operatorCompatibleType
	// operatorValueIterableType is for iterating over values.
	operatorValueIterableType
	operatorKeyIterableType

	nameTempVar string = "__TMP_VAR_TO_RECHECK_"
)

func init() {
	if typeAny == nil {
		panic("initialization of 'any' type failed")
	}

	if typeError == nil {
		panic("initialization of 'error' type failed")
	}

	TemplateManager = NewWorkspaceTemplateManager()
}

// SetCustomFunctions sets the custom template functions for the workspace.
// These functions are discovered by scanning Go source files for template.FuncMap definitions.
// They will be merged with builtin functions during analysis.
func SetCustomFunctions(funcs map[string]*FunctionDefinition) {
	customFunctions = funcs
}

// GetCustomFunctions returns the currently set custom template functions.
func GetCustomFunctions() map[string]*FunctionDefinition {
	return customFunctions
}

func NewFileDefinition(
	fileName string,
	root *parser.GroupStatementNode,
	outterTemplate map[*parser.GroupStatementNode]*TemplateDefinition,
) (*FileDefinition, map[string]*VariableDefinition, map[string]*VariableDefinition) {
	file := new(FileDefinition)

	file.name = fileName
	file.root = root
	file.isTemplateGroupAlreadyAnalyzed = false

	file.typeHints = make(map[*parser.GroupStatementNode]types.Type)
	file.templates = make(map[string]*TemplateDefinition)
	file.extraVariableNameWithTypeInferenceBehavior = make(map[string]*VariableDefinition)
	file.secondaryVariable = nil

	// 2. build external templates available for the current file
	foundMoreThanOnce := make(map[string]bool)

	for templateNode, templateDef := range outterTemplate {
		templateName := templateNode.TemplateName()

		def := file.templates[templateName]

		if def != nil {
			if !foundMoreThanOnce[templateName] {
				defAny := &TemplateDefinition{
					inputType: typeAny.Type(),
					fileName:  "",
					node:      nil,
				}

				file.templates[templateName] = defAny
			}

			foundMoreThanOnce[templateName] = true
			continue
		}

		file.templates[templateName] = templateDef
		foundMoreThanOnce[templateName] = false
	}

	file.functions = getBuiltinFunctionDefinition()
	// Merge custom functions (from template.FuncMap) with builtins
	// Custom functions don't override builtins
	for name, def := range customFunctions {
		if _, exists := file.functions[name]; !exists {
			file.functions[name] = def
		}
	}
	file.scopeToVariables = make(
		map[*parser.GroupStatementNode]map[string]*VariableDefinition,
	)

	globalVariables, localVariables := NewGlobalAndLocalVariableDefinition(
		nil,
		root,
		fileName,
	)

	return file, globalVariables, localVariables
}

func NewFileDefinitionFromPartialFile(
	partialFile *FileDefinition,
	outterTemplate map[*parser.GroupStatementNode]*TemplateDefinition,
) (*FileDefinition, map[string]*VariableDefinition, map[string]*VariableDefinition) {
	if partialFile == nil {
		log.Printf("got a <nil> partial File\n")
		panic("got a <nil> partial File")
	}

	if partialFile.root == nil {
		log.Printf(
			"partial file without root parse tree found at start definition analysis"+
				"\n fileName = %s\n",
			partialFile.FileName(),
		)
		panic("partial file without root parse tree found at start definition analysis")
	}

	if partialFile.name == "" {
		panic("partial file cannot have empty file name")
	}

	file, globalVariables, localVariables := NewFileDefinition(
		partialFile.FileName(),
		partialFile.root,
		outterTemplate,
	)
	file.root = partialFile.root
	file.functions = partialFile.functions
	file.scopeToVariables = partialFile.scopeToVariables

	return file, globalVariables, localVariables
}

// getBuiltinFunctionDefinition returns the built-in template functions (and, or, len, etc.).
func getBuiltinFunctionDefinition() map[string]*FunctionDefinition {
	anyType := typeAny.Type()
	boolType := types.Typ[types.Bool]
	intType := types.Typ[types.Int]
	stringType := types.Typ[types.String]

	// Helper to create a variadic function signature: func(args ...any) returnType
	variadicAny := func(returnType types.Type) *types.Signature {
		anySlice := types.NewSlice(anyType)
		params := types.NewTuple(types.NewVar(token.NoPos, nil, "args", anySlice))
		results := types.NewTuple(types.NewVar(token.NoPos, nil, "", returnType))
		return types.NewSignatureType(nil, nil, nil, params, results, true)
	}

	// Helper to create a single-arg function: func(arg any) returnType
	singleAny := func(returnType types.Type) *types.Signature {
		params := types.NewTuple(types.NewVar(token.NoPos, nil, "arg", anyType))
		results := types.NewTuple(types.NewVar(token.NoPos, nil, "", returnType))
		return types.NewSignatureType(nil, nil, nil, params, results, false)
	}

	// Define builtin function signatures
	builtinSignatures := map[string]*types.Signature{
		// Logical functions - variadic, return any (last/first non-empty arg)
		"and": variadicAny(anyType),
		"or":  variadicAny(anyType),

		// Negation - single arg, returns bool
		"not": singleAny(boolType),

		// Length - single arg, returns int
		"len": singleAny(intType),

		// Print functions - variadic, return string
		"print":   variadicAny(stringType),
		"printf":  variadicAny(stringType),
		"println": variadicAny(stringType),

		// Escape functions - single arg, return string
		"html":     singleAny(stringType),
		"js":       singleAny(stringType),
		"urlquery": singleAny(stringType),

		// Indexing/slicing - variadic, return any
		"index": variadicAny(anyType),
		"slice": variadicAny(anyType),

		// Call - variadic (first arg is function), return any
		"call": variadicAny(anyType),

		// Comparison functions - variadic (for eq/ne which can take 2+ args), return bool
		"eq": variadicAny(boolType),
		"ne": variadicAny(boolType),
		"lt": variadicAny(boolType),
		"le": variadicAny(boolType),
		"gt": variadicAny(boolType),
		"ge": variadicAny(boolType),
	}

	// Boolean constants and control flow - these are not really functions but
	// are listed in the symbol table. Give them a zero-arg signature returning appropriate type.
	zeroArgBool := func() *types.Signature {
		results := types.NewTuple(types.NewVar(token.NoPos, nil, "", boolType))
		return types.NewSignatureType(nil, nil, nil, nil, results, false)
	}
	zeroArgVoid := func() *types.Signature {
		return types.NewSignatureType(nil, nil, nil, nil, nil, false)
	}

	builtinSignatures["true"] = zeroArgBool()
	builtinSignatures["false"] = zeroArgBool()
	builtinSignatures["continue"] = zeroArgVoid()
	builtinSignatures["break"] = zeroArgVoid()

	builtinFunctionDefinition := make(map[string]*FunctionDefinition)

	for name, sig := range builtinSignatures {
		def := &FunctionDefinition{}
		def.name = name
		def.node = nil
		def.fileName = "builtin"
		def.typ = sig

		builtinFunctionDefinition[name] = def
	}

	return builtinFunctionDefinition
}

func NewGlobalAndLocalVariableDefinition(
	node parser.AstNode,
	parent *parser.GroupStatementNode,
	fileName string,
) (map[string]*VariableDefinition, map[string]*VariableDefinition) {
	globalVariables := make(map[string]*VariableDefinition)
	localVariables := make(map[string]*VariableDefinition)

	localVariables["."] = NewVariableDefinition(".", nil, parent, fileName)
	localVariables["$"] = NewVariableDefinition("$", nil, parent, fileName)

	return globalVariables, localVariables
}

func NewVariableDefinition(
	variableName string,
	node parser.AstNode,
	parent *parser.GroupStatementNode,
	fileName string,
) *VariableDefinition {
	def := &VariableDefinition{}

	def.name = variableName
	def.parent = parent
	def.fileName = fileName

	def.TreeImplicitType = nil
	def.typ = typeAny.Type()

	if node != nil {
		def.node = node
		def.rng = node.Range()
	}

	return def
}

func cloneVariableDefinition(old *VariableDefinition) *VariableDefinition {
	fresh := &VariableDefinition{}

	fresh.name = old.name
	fresh.fileName = old.fileName

	fresh.typ = old.typ
	fresh.node = old.node
	fresh.parent = old.parent
	fresh.rng = old.rng

	fresh.IsUsedOnce = old.IsUsedOnce
	fresh.TreeImplicitType = old.TreeImplicitType

	return fresh
}

func DefinitionAnalysisFromPartialFile(
	partialFile *FileDefinition,
	outterTemplate map[*parser.GroupStatementNode]*TemplateDefinition,
) (*FileDefinition, []lexer.Error) {
	if partialFile == nil {
		log.Printf(
			"expected a partial file but got <nil> for 'DefinitionAnalysisFromPartialFile()'",
		)
		panic(
			"expected a partial file but got <nil> for 'DefinitionAnalysisFromPartialFile()'",
		)
	}

	file, globalVariables, localVariables := NewFileDefinitionFromPartialFile(
		partialFile,
		outterTemplate,
	)
	file.isTemplateGroupAlreadyAnalyzed = true

	typ, _, errs := definitionAnalysisRecursive(
		file.root,
		nil,
		file,
		globalVariables,
		localVariables,
	)

	_ = typ
	// file.typeHints[file.root] = typ[0]

	return file, errs
}

func DefinitionAnalysis(
	fileName string,
	node *parser.GroupStatementNode,
	outterTemplate map[*parser.GroupStatementNode]*TemplateDefinition,
) (*FileDefinition, []lexer.Error) {
	if node == nil {
		return nil, nil
	}

	fileInfo, globalVariables, localVariables := NewFileDefinition(
		fileName,
		node,
		outterTemplate,
	)

	_, _, errs := definitionAnalysisRecursive(
		node,
		nil,
		fileInfo,
		globalVariables,
		localVariables,
	)

	return fileInfo, errs
}

func definitionAnalysisRecursive(
	node parser.AstNode,
	parent *parser.GroupStatementNode,
	file *FileDefinition,
	globalVariables, localVariables map[string]*VariableDefinition,
) ([2]types.Type, InferenceFoundReturn, []lexer.Error) {
	if globalVariables == nil || localVariables == nil {
		panic(
			"arguments global/local variable definition for 'definitionAnalysis()' shouldn't be 'nil'",
		)
	}

	var errs []lexer.Error
	var localInferences InferenceFoundReturn
	var statementType [2]types.Type

	switch n := node.(type) {
	case *parser.GroupStatementNode:
		statementType, localInferences, errs = definitionAnalysisGroupStatement(
			n,
			parent,
			file,
			globalVariables,
			localVariables,
		)

	case *parser.TemplateStatementNode:
		statementType, localInferences, errs = definitionAnalysisTemplatateStatement(
			n,
			parent,
			file,
			globalVariables,
			localVariables,
		)

	case *parser.CommentNode:
		statementType, localInferences, errs = definitionAnalysisComment(
			n,
			parent,
			file,
			globalVariables,
			localVariables,
		)

	case *parser.VariableDeclarationNode:
		statementType, localInferences, errs = definitionAnalysisVariableDeclaration(
			n,
			parent,
			file,
			globalVariables,
			localVariables,
		)

	case *parser.VariableAssignationNode:
		statementType, localInferences, errs = definitionAnalysisVariableAssignment(
			n,
			parent,
			file,
			globalVariables,
			localVariables,
		)

	case *parser.MultiExpressionNode:
		statementType, localInferences, errs = definitionAnalysisMultiExpression(
			n,
			parent,
			file,
			globalVariables,
			localVariables,
		)

	case *parser.ExpressionNode:
		statementType, localInferences, errs = definitionAnalysisExpression(
			n,
			parent,
			file,
			globalVariables,
			localVariables,
		)

	case *parser.SpecialCommandNode:
		// nothing to analyze here

	default:
		panic("unknown parseNode found. node type = " + node.Kind().String())
	}

	return statementType, localInferences, errs
}
