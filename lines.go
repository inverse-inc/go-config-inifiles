package inifiles

// splitLines splits data into lines, detecting the line ending from the first
// line (mirroring Perl's _nextline). It returns the lines (with terminators
// stripped) and the detected line ending (nil if the data contains no line
// terminator at all, which the parser treats as zero lines). A leading UTF-8
// BOM is stripped from the start of the data.
func splitLines(data []byte) ([]string, []byte) {
	// Strip UTF-8 BOM if present.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	// Find the first line terminator.
	i := 0
	for i < len(data) {
		switch data[i] {
		case 0x0D, 0x0A, 0x15:
			goto found
		}
		i++
	}
	// No terminator at all: Perl's first read returns undef -> zero lines.
	return nil, nil

found:
	var lineEnds []byte
	termLen := 1
	if data[i] == 0x0D {
		if i+1 < len(data) && data[i+1] == 0x0A {
			lineEnds = []byte{0x0D, 0x0A}
			termLen = 2
		} else {
			lineEnds = []byte{0x0D}
		}
	} else if data[i] == 0x0A {
		lineEnds = []byte{0x0A}
	} else {
		lineEnds = []byte{0x15}
	}

	lines := []string{string(data[:i])}
	pos := i + termLen
	start := pos
	for pos < len(data) {
		switch data[pos] {
		case 0x0D:
			if pos+1 < len(data) && data[pos+1] == 0x0A {
				lines = append(lines, string(data[start:pos]))
				pos += 2
				start = pos
			} else {
				lines = append(lines, string(data[start:pos]))
				pos++
				start = pos
			}
		case 0x0A, 0x15:
			lines = append(lines, string(data[start:pos]))
			pos++
			start = pos
		default:
			pos++
		}
	}
	// Append a trailing partial line (content after the final terminator).
	if start < len(data) {
		lines = append(lines, string(data[start:]))
	}
	return lines, lineEnds
}