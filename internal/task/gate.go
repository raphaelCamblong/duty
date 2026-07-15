package task

import (
	"bytes"
	"fmt"
	"strings"
)

// gatesHeading names the section the gate checklist lives under.
const gatesHeading = "Gates"

// boxIndex is the offset of the checkbox character within a "- [ ]" gate line.
const boxIndex = len("- [")

// Gate is one gate checkbox: its text and whether it is ticked.
type Gate struct {
	// Text is the gate line's text after the checkbox.
	Text string
	// Done reports whether the checkbox is ticked.
	Done bool
}

// Gates returns the gate checkboxes under the "## Gates" section in file order.
func Gates(content []byte) []Gate {
	body, ok := Section(content, gatesHeading)
	if !ok {
		return nil
	}
	var gates []Gate
	for _, raw := range strings.Split(string(body), "\n") {
		if g, ok := parseGate(strings.TrimRight(raw, " \t\r")); ok {
			gates = append(gates, g)
		}
	}
	return gates
}

// AddGate appends a "- [ ] <text>" gate to the "## Gates" section, creating the
// section like ReplaceSection when it is absent. Existing gate lines survive
// byte-for-byte; the new gate goes after the last one.
func AddGate(content []byte, text string) []byte {
	line := "- [ ] " + strings.TrimSpace(text)
	start, end, found := sectionBounds(content, gatesHeading)
	if !found {
		out, _ := ReplaceSection(content, gatesHeading, []byte(line))
		return out
	}
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
	start, end, found := sectionBounds(content, gatesHeading)
	count := 0
	if found {
		for pos := start; pos < end; {
			line, next := lineAt(content, pos)
			if _, ok := parseGate(strings.TrimRight(string(line), " \t\r")); ok {
				count++
				if count == n {
					return flipBox(content, pos+boxIndex, done), nil
				}
			}
			pos = next
		}
	}
	return nil, fmt.Errorf("no gate %d: task has %d", n, count)
}

// parseGate decodes a right-trimmed line as a gate; ok is false for any other
// line. Indented checkboxes do not match, mirroring CountGates.
func parseGate(line string) (Gate, bool) {
	switch {
	case strings.HasPrefix(line, "- [x]"):
		return Gate{Text: strings.TrimSpace(line[len("- [x]"):]), Done: true}, true
	case strings.HasPrefix(line, "- [ ]"):
		return Gate{Text: strings.TrimSpace(line[len("- [ ]"):]), Done: false}, true
	}
	return Gate{}, false
}

// lastGateEnd returns the offset just past the last gate line within [start,end)
// and whether the section holds any gate.
func lastGateEnd(content []byte, start, end int) (int, bool) {
	at, ok := 0, false
	for pos := start; pos < end; {
		line, next := lineAt(content, pos)
		if _, isGate := parseGate(strings.TrimRight(string(line), " \t\r")); isGate {
			at, ok = next, true
		}
		pos = next
	}
	return at, ok
}

// flipBox returns content with the checkbox byte at at set ticked or unticked.
func flipBox(content []byte, at int, done bool) []byte {
	box := byte(' ')
	if done {
		box = 'x'
	}
	out := bytes.Clone(content)
	out[at] = box
	return out
}
