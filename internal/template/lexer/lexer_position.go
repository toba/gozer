package lexer

import "log"

// ConvertSingleIndexToTextEditorPosition converts a character index to a text editor position.
func ConvertSingleIndexToTextEditorPosition(buffer []byte, charIndex int) Position {
	var line, col int

	for i := range buffer {
		if i == charIndex {
			break
		}

		if buffer[i] == byte('\n') {
			line++
			col = 0
		} else {
			col++
		}
	}

	pos := Position{Line: line, Character: col}

	return pos
}

// 'rangeIndex' is includes at rangeIndex[0] but exclusive at rangeIndex[1] (eg. [a, b[)
func convertRangeIndexToTextEditorPosition(
	editorContent []byte,
	rangeIndex []int,
	initialLine, initialColumn int,
) Range {
	if rangeIndex[0] > rangeIndex[1] {
		log.Printf(
			"bad range formating.\n start = '%d' :: end = '%d'\n",
			rangeIndex[0],
			rangeIndex[1],
		)
		panic("bad range formating, 'end position' cannot be before 'start position'")
	}

	if rangeIndex[0] == rangeIndex[1] {
		return Range{}
	}

	position := Range{}
	position.Start = ConvertSingleIndexToTextEditorPosition(editorContent, rangeIndex[0])
	position.End = ConvertSingleIndexToTextEditorPosition(editorContent, rangeIndex[1]-1)

	if position.Start.Line == 0 {
		position.Start.Character += initialColumn
	}

	if position.End.Line == 0 {
		position.End.Character += initialColumn
	}

	position.Start.Line += initialLine
	position.End.Line += initialLine

	return position
}
