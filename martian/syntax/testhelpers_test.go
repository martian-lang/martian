// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Helper and setup methods for tests

package syntax

import (
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/martian-lang/martian/martian/util"
)

type nullWriter struct{}

func (*nullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
func (*nullWriter) WriteString(b string) (int, error) {
	return len(b), nil
}

var devNull nullWriter

func TestMain(m *testing.M) {
	SetEnforcementLevel(EnforceError)
	// Disable logging here, because otherwise the race detector can get unhappy
	// when running parallel tests.
	util.SetPrintLogger(&devNull)
	util.LogTeeWriter(&devNull)
	os.Exit(m.Run())
}

// Checks that the source can be parsed and compiled.
func testGood(t *testing.T, src string) *Ast {
	t.Helper()
	if ast, err := yaccParse([]byte(src), new(SourceFile),
		makeStringIntern()); err != nil {
		t.Fatal(err.Error())
		return nil
	} else if err := ast.compile(); err != nil {
		t.Errorf("Failed to compile src: %v\n%s", err, err.Error())
		return nil
	} else {
		return ast
	}
}

// Checks that the source can be parsed but does not compile.
func testBadCompile(t *testing.T, src, expect string) {
	t.Helper()
	if ast, err := yaccParse([]byte(src), new(SourceFile), makeStringIntern()); err != nil {
		t.Fatal(err.Error())
		return
	} else if err := ast.compile(); err == nil {
		t.Error("Expected failure to compile.")
		return
	} else {
		msg := err.Error()
		if !strings.Contains(msg, expect) {
			t.Errorf("Expected %q, got %q", expect, msg)
		}
		return
	}
}

// Checks that the source cannot be parsed.
func testBadGrammar(t *testing.T, src string) {
	t.Helper()
	if _, err := yaccParse([]byte(src), new(SourceFile), makeStringIntern()); err == nil {
		t.Error("Expected failure to parse, but got success.")
	}
}

// Produce a relatively debuggable side-by-side diff.
func diffLines(src, formatted string, t *testing.T) {
	t.Helper()
	src_lines := strings.Split(src, "\n")
	formatted_lines := strings.Split(formatted, "\n")
	offset := 0
	// Replace tabs with \t for visibility, and truncate lines over 30
	// characters so that the diff will fit.
	trimLine := func(line string) string {
		line = strings.Replace(line, "\t", "\\t", -1)
		if len(line) > 30 {
			line = line[:27] + "..."
		}
		return line
	}
	removeSpace := func(line string) string {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			return ""
		}
		flds := strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || unicode.IsSpace(r)
		})
		if len(flds) == 1 {
			return flds[0]
		}
		return strings.Join(flds, "")
	}
	// Prevent overwhelming the console output
	firstWrongLine := 0
	for i, line := range src_lines {
		if i >= len(formatted_lines) || line != formatted_lines[i] {
			firstWrongLine = i
			break
		}
	}
	wrongLines := 0
	lastWrongLine := 0
	// Used to search for anchor lines.  If a line is unique in both sets,
	// then we should match them up in the diff
	var uniqueSrcLines, uniqueFmtLines map[string]int
	for i, line := range src_lines {
		if i < firstWrongLine-20 {
			continue
		}
		pad := ""
		if fmtLen := len(line) + strings.Count(line, "\t"); fmtLen < 30 {
			// Add the tab count because when formatting we'll replace those
			// characters with \t.
			pad = strings.Repeat(" ", 30-fmtLen)
		}
		if len(formatted_lines) > i+offset {
			if line == formatted_lines[i+offset] {
				line = trimLine(line)
				t.Logf("%3d: %s %s= %s", i, line, pad, line)
				if wrongLines > 20 || wrongLines > 0 && lastWrongLine < i-10 {
					t.Logf("...")
					return
				}
			} else if cline := removeSpace(line); cline == removeSpace(formatted_lines[i+offset]) {
				if strings.TrimLeftFunc(
					line, unicode.IsSpace) == strings.TrimLeftFunc(
					formatted_lines[i+offset], unicode.IsSpace) {
					t.Errorf("%3d: %s %s| %s", i,
						trimLine(line), pad, trimLine(formatted_lines[i+offset]))
					wrongLines++
					lastWrongLine = i
				} else {
					t.Errorf("%3d: %s %s| %s", i,
						line, pad, formatted_lines[i+offset])
					wrongLines++
					lastWrongLine = i
				}
			} else if len(cline) == 0 {
				t.Errorf("%3d: %s %s<", i, strings.Replace(line, "\t", "\\t", -1), pad)
				wrongLines++
				lastWrongLine = i
				offset--
			} else {
				// Look one line ahead or behind
				if len(formatted_lines) > i+offset+1 {
					if formatted_lines[i+offset+1] == line {
						t.Errorf("%s > %s", strings.Repeat(" ", 35),
							formatted_lines[i+offset])
						wrongLines++
						offset++
						lastWrongLine = i
						line = trimLine(line)
						t.Logf("%3d: %s %s= %s", i, line, pad, line)
						continue
					}
				}
				if src_lines[i+1] == formatted_lines[i+offset] {
					t.Errorf("%3d: %s %s<", i, line, pad)
					wrongLines++
					lastWrongLine = i
					offset--
					continue
				}
				// Try to find the next unique source line which matches.
				if uniqueSrcLines == nil {
					uniqueSrcLines = make(map[string]int, (len(src_lines)-i)/2)
					for j, line := range src_lines[i:] {
						if _, ok := uniqueSrcLines[line]; ok {
							uniqueSrcLines[line] = -1
						} else {
							uniqueSrcLines[line] = i + j
						}
					}
				}
				if uniqueFmtLines == nil {
					uniqueFmtLines = make(map[string]int, (len(formatted_lines)-i-offset)/2)
					for j, line := range formatted_lines[i+offset:] {
						if _, ok := uniqueFmtLines[line]; ok {
							uniqueFmtLines[line] = -1
						} else {
							uniqueFmtLines[line] = j + i + offset
						}
					}
				}
				if j, ok := uniqueSrcLines[line]; ok && j >= 0 {
					if k, ok := uniqueFmtLines[line]; ok && k >= 0 {
						for k > i+offset {
							t.Errorf("%s > %s", strings.Repeat(" ", 35),
								formatted_lines[i+offset])
							wrongLines++
							offset++
							lastWrongLine = i
						}
						line = trimLine(line)
						t.Logf("%3d: %s %s= %s", i, line, pad, line)
						continue
					} else if k == -1 {
						uniqueFmtLines = nil
					}
				} else if j == -1 {
					uniqueSrcLines = nil
				}
				forwardOffset := 0
				for moreOffset, fline := range formatted_lines[i+offset:] {
					if removeSpace(fline) == cline {
						forwardOffset = moreOffset
						break
					} else if moreOffset > 20 {
						break
					}
				}
				backwardOffset := 0
				cline := removeSpace(formatted_lines[i+offset])
				for moreOffset, uline := range src_lines[i:] {
					if removeSpace(uline) == cline {
						backwardOffset = moreOffset
						break
					} else if moreOffset > 20 {
						break
					}
				}
				if forwardOffset == 0 && backwardOffset == 0 {
					t.Errorf("%3d: %s %s| %s", i, line, pad, formatted_lines[i+offset])
					wrongLines++
					lastWrongLine = i
				} else if (forwardOffset == 0 && backwardOffset != 0) ||
					(backwardOffset > forwardOffset) {
					t.Errorf("%3d: %s %s<", i, line, pad)
					offset--
					wrongLines++
					lastWrongLine = i
				} else {
					for j := 0; j < forwardOffset; j++ {
						t.Errorf("%s > %s", strings.Repeat(" ", 35), formatted_lines[i+j+offset])
						wrongLines++
						lastWrongLine = i
					}
					offset += forwardOffset
					if line == formatted_lines[i+offset] {
						line = trimLine(line)
						t.Logf("%3d: %s %s= %s", i, line, pad, line)
					} else {
						t.Errorf("%3d: %s %s| %s", i, line, pad, formatted_lines[i+offset])
						wrongLines++
						lastWrongLine = i
					}
				}
			}
		} else {
			t.Errorf("%3d: %s %s<", i, line, pad)
			wrongLines++
			if wrongLines > 30 {
				t.Logf("...")
				return
			}
		}
	}
	if len(formatted_lines) > len(src_lines)+offset {
		for _, line := range formatted_lines[len(src_lines)+offset:] {
			t.Errorf("%s > %s", strings.Repeat(" ", 35), line)
		}
	}
}
