package task

import (
	"bytes"
	"fmt"
	"strings"
)

const gatesHeading = "Gates"

// boxIndex is the offset of the checkbox character within a "- [ ]" gate line.
const boxIndex = len("- [")

type Gate struct {
	// Text is the gate line's text after the checkbox.
	Text string
	Done bool
}

func Gates(content []byte) []Gate {
	body, ok := Section(content, gatesHeading)
	if !ok {
		return nil
	}
	var gates []Gate
	for _, raw := range strings.Split(string(body), "\n") {
		if g, ok := parseGate(raw); ok {
			gates = append(gates, g)
		}
	}
	return gates
}

// AddGates appends a "- [ ] <text>" gate per text, in order, to the "## Gates"
// section, creating the section on the first when it is absent. Existing gate
// lines survive byte-for-byte; each new gate goes after the last.
func AddGates(content []byte, texts []string) []byte {
	for _, text := range texts {
		content = AddGate(content, text)
	}
	return content
}

// AddGate appends a "- [ ] <text>" gate to the "## Gates" section, creating the
// section like ReplaceSection when it is absent. Existing gate lines survive
// byte-for-byte; the new gate goes after the last one.
func AddGate(content []byte, text string) []byte {
	line := "- [ ] " + strings.TrimSpace(text)
	start, end, _ := sectionBounds(content, gatesHeading)
	at, ok := lastGateEnd(content, start, end)
	if !ok {
		out, _ := ReplaceSection(content, gatesHeading, []byte(line))
		return out
	}
	ins := line + "\n"
	if content[at-1] != '\n' {
		ins = "\n" + ins
	}
	return splice(content, at, at, []byte(ins))
}

// SetGate sets the ticked state of the n-th gate (1-based) under "## Gates",
// flipping only that line's checkbox character. It errors when n is out of
// range for the gates present.
func SetGate(content []byte, n int, done bool) ([]byte, error) {
	start, end, _ := sectionBounds(content, gatesHeading)
	positions := gatePositions(content, start, end)
	if n < 1 || n > len(positions) {
		return nil, fmt.Errorf("no gate %d: task has %d", n, len(positions))
	}
	return flipBox(content, positions[n-1]+boxIndex, done), nil
}

// SetAllGates ticks or unticks every gate under "## Gates", flipping only each
// gate line's checkbox character. Content with no Gates section is returned
// unchanged.
func SetAllGates(content []byte, done bool) []byte {
	start, end, found := sectionBounds(content, gatesHeading)
	if !found {
		return content
	}
	out := bytes.Clone(content)
	for _, pos := range gatePositions(content, start, end) {
		out[pos+boxIndex] = boxByte(done)
	}
	return out
}

// parseGate decodes a line as a gate; ok is false for anything else, including
// an indented checkbox (only a top-level "- [ ]"/"- [x]" is a gate).
func parseGate(line string) (Gate, bool) {
	switch {
	case strings.HasPrefix(line, "- [x]"):
		return Gate{Text: strings.TrimSpace(line[len("- [x]"):]), Done: true}, true
	case strings.HasPrefix(line, "- [ ]"):
		return Gate{Text: strings.TrimSpace(line[len("- [ ]"):]), Done: false}, true
	}
	return Gate{}, false
}

func lastGateEnd(content []byte, start, end int) (int, bool) {
	positions := gatePositions(content, start, end)
	if len(positions) == 0 {
		return 0, false
	}
	_, next := lineAt(content, positions[len(positions)-1])
	return next, true
}

func gatePositions(content []byte, start, end int) []int {
	var positions []int
	for pos := start; pos < end; {
		line, next := lineAt(content, pos)
		if _, ok := parseGate(string(line)); ok {
			positions = append(positions, pos)
		}
		pos = next
	}
	return positions
}

func flipBox(content []byte, at int, done bool) []byte {
	out := bytes.Clone(content)
	out[at] = boxByte(done)
	return out
}

func boxByte(done bool) byte {
	if done {
		return 'x'
	}
	return ' '
}
