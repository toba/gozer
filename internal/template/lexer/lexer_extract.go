package lexer

import (
	"bytes"
	"log"
	"regexp"
)

// templateExtractor holds state while scanning source code for {{...}} blocks.
type templateExtractor struct {
	content       []byte
	currentLine   int
	currentColumn int
	codes         [][]byte
	positions     []Range
}

// processLoneDelimiter handles an unmatched {{ or }} delimiter (a syntax error case).
// Returns the next lone delimiter location after advancing past this one.
func (e *templateExtractor) processLoneDelimiter(
	loneLoc []int,
	lonePattern *regexp.Regexp,
) []int {
	templatePosition := convertRangeIndexToTextEditorPosition(
		e.content,
		loneLoc,
		e.currentLine,
		e.currentColumn,
	)

	e.positions = append(e.positions, templatePosition)
	e.codes = append(e.codes, e.content[loneLoc[0]:loneLoc[1]])

	e.currentLine = templatePosition.End.Line
	e.currentColumn = templatePosition.End.Character + 1
	e.content = e.content[loneLoc[1]:]

	return lonePattern.FindIndex(e.content)
}

// extractTemplateCode scans source for all {{...}} template blocks and returns
// their inner content (without delimiters) along with their source positions.
func extractTemplateCode(content []byte) ([][]byte, []Range) {
	if len(content) == 0 {
		return nil, nil
	}

	var ORIGINAL_CONTENT = content
	var CLONED_CONTENT = bytes.Clone(content)
	content = CLONED_CONTENT

	// Use pre-compiled patterns
	loneDelimPattern := compiledPatterns.loneDelimiter
	templatePattern := compiledPatterns.templateStatement

	ext := &templateExtractor{
		content:       content,
		currentLine:   0,
		currentColumn: 0,
		codes:         nil,
		positions:     nil,
	}

	var loc, loneLoc []int
	var templatePosition Range

	for {
		loneLoc = loneDelimPattern.FindIndex(ext.content)
		loc = templatePattern.FindIndex(ext.content)

		if loc == nil {
			// Process remaining lone delimiters
			for loneLoc != nil {
				loneLoc = ext.processLoneDelimiter(loneLoc, loneDelimPattern)
			}
			break
		}

		// Process lone delimiters that appear before the template statement
		for loneLoc != nil && loneLoc[0] < loc[0] {
			loneLoc = ext.processLoneDelimiter(loneLoc, loneDelimPattern)
			loc = templatePattern.FindIndex(ext.content)
		}

		templatePosition = convertRangeIndexToTextEditorPosition(
			ext.content,
			loc,
			ext.currentLine,
			ext.currentColumn,
		)

		ext.currentLine = templatePosition.End.Line
		ext.currentColumn = templatePosition.End.Character + 1

		// Extract content between {{ and }}
		insideTemplate := ext.content[loc[0]+2 : loc[1]-2]

		templatePosition.Start.Character += 2
		templatePosition.End.Character -= 1

		ext.positions = append(ext.positions, templatePosition)
		ext.codes = append(ext.codes, insideTemplate)

		ext.content = ext.content[loc[1]:]
	}

	if !bytes.Equal(ORIGINAL_CONTENT, CLONED_CONTENT) {
		log.Printf(
			"ORIGINAL_CONTENT = \n%q\n===================\ncontent = \n%q\n=============",
			ORIGINAL_CONTENT,
			CLONED_CONTENT,
		)
		panic(
			"content of the file has changed during lexical analysis (extracting template)." +
				"In a perfect world, it shouldn't change",
		)
	}

	return ext.codes, ext.positions
}
