package analyzer

import (
	"errors"
	"go/scanner"
	"go/token"
	"go/types"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

func remapRangeFromCommentGoCodeToSource(
	header string,
	boundary, target lexer.Range,
) lexer.Range {
	maxLineInHeader := 0

	for _, char := range []byte(header) {
		if char == '\n' {
			maxLineInHeader++
		}
	}

	rangeRemaped := lexer.Range{}

	// NOTE: because of go/parser, 'target.Start.Line' always start at '1'
	rangeRemaped.Start.Line = boundary.Start.Line + target.Start.Line - 1 - maxLineInHeader
	rangeRemaped.End.Line = boundary.Start.Line + target.End.Line - 1 - maxLineInHeader

	rangeRemaped.Start.Character = target.Start.Character - 1
	rangeRemaped.End.Character = target.End.Character - 1

	if target.Start.Line-maxLineInHeader == 1 {
		rangeRemaped.Start.Character = boundary.Start.Character + len(
			header,
		) + target.Start.Character
	}

	if target.End.Line-maxLineInHeader == 1 {
		rangeRemaped.End.Character = boundary.Start.Character + len(
			header,
		) + target.End.Character
	}

	if rangeRemaped.End.Line > boundary.End.Line {
		// msg := "boundary.End.Line = %d ::: rangeRemaped.End.Line = %d\n"
		// log.Printf(msg, boundary.End.Line, rangeRemaped.End.Line)
		// log.Printf("boundary.End.Line = %d ::: rangeRemaped.End.Line = %d\n", boundary.End.Line, rangeRemaped.End.Line)

		rangeRemaped.End.Line = boundary.End.Line
		// panic("remaped range cannot excede the comment GoCode boundary")
	}

	return rangeRemaped
}

func goAstPositionToRange(startPos, endPos token.Position) lexer.Range {
	distance := lexer.Range{
		Start: lexer.Position{
			Line:      startPos.Line,
			Character: startPos.Column,
		},
		End: lexer.Position{
			Line:      endPos.Line,
			Character: endPos.Column,
		},
	}

	return distance
}

func NewParseErrorFromErrorType(err types.Error) *parser.ParseError {
	fset := err.Fset
	pos := fset.Position(err.Pos)

	parseErr := parser.ParseError{
		Err: errors.New(err.Msg),
		Range: lexer.Range{
			Start: lexer.Position{
				Line:      pos.Line,
				Character: pos.Column,
			},
			End: lexer.Position{
				Line:      pos.Line,
				Character: pos.Column + pos.Offset - 1,
			},
		},
	}

	return &parseErr
}

func NewParseErrorFromErrorList(
	err *scanner.Error,
	randomColumnOffset int,
) *parser.ParseError {
	if err == nil {
		return nil
	}

	if randomColumnOffset < 0 {
		randomColumnOffset = 10
	}

	parseErr := &parser.ParseError{
		Err: errors.New(err.Msg),
		Range: lexer.Range{
			Start: lexer.Position{
				Line:      err.Pos.Line,
				Character: err.Pos.Column,
			},
			End: lexer.Position{
				Line:      err.Pos.Line,
				Character: err.Pos.Column + randomColumnOffset - 1,
			},
		},
	}

	return parseErr
}

func convertThirdPartiesParseErrorToLocalError(
	parseError error,
	errsType []types.Error,
	file *FileDefinition,
	comment *parser.CommentNode,
	virtualHeader string,
) []lexer.Error {
	if file == nil {
		panic("file definition cannot be 'nil' while convert std error to project error")
	}

	if comment == nil {
		panic("comment node cannot be 'nil' while convert std error to project error")
	}

	var errs []lexer.Error

	// 1. convert parse error from go/ast.Error to lexer.Error, and adjust the 'Range'

	if parseError != nil {
		// log.Println("comment scanner error found, ", parseError)

		var errorList scanner.ErrorList
		ok := errors.As(parseError, &errorList)
		if !ok {
			panic(
				"unexpected error, error obtained by go code parsing did not return expected type ('scanner.ErrorList')",
			)
		}

		const randomColumnOffset int = 7

		for _, errScanner := range errorList {
			// A. Build diagnostic errors
			parseErr := NewParseErrorFromErrorList(errScanner, randomColumnOffset)
			parseErr.Range = remapRangeFromCommentGoCodeToSource(
				virtualHeader,
				comment.GoCode.Range,
				parseErr.Range,
			)

			// log.Println("comment scanner error :: ", parseErr)

			errs = append(errs, parseErr)
		}
	}

	// 2. convert type error to lexer.Error

	for _, err := range errsType {
		parseErr := NewParseErrorFromErrorType(err)
		parseErr.Range = remapRangeFromCommentGoCodeToSource(
			virtualHeader,
			comment.GoCode.Range,
			parseErr.Range,
		)

		errs = append(errs, parseErr)
	}

	return errs
}
