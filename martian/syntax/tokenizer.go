// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

// Martian tokenizer. Simple regexp-based implementation.

package syntax

import (
	"regexp"
)

const default_out_name = "default"

// re matches text to produce token.
type rule struct {
	re    *regexp.Regexp
	tokid int
}

var rules = [...]rule{
	// Order matters.
	{regexp.MustCompile(`^\s+`), SKIP},      // whitespace
	{regexp.MustCompile(`^#.*\n`), COMMENT}, // Python-style comments
	{regexp.MustCompile(`^@include`), INCLUDE_DIRECTIVE},
	{regexp.MustCompile(`^=`), EQUALS},
	{regexp.MustCompile(`^\(`), LPAREN},
	{regexp.MustCompile(`^\)`), RPAREN},
	{regexp.MustCompile(`^{`), LBRACE},
	{regexp.MustCompile(`^}`), RBRACE},
	{regexp.MustCompile(`^\[`), LBRACKET},
	{regexp.MustCompile(`^\]`), RBRACKET},
	{regexp.MustCompile(`^:`), COLON},
	{regexp.MustCompile(`^;`), SEMICOLON},
	{regexp.MustCompile(`^,`), COMMA},
	{regexp.MustCompile(`^\.`), DOT},
	{regexp.MustCompile(`^"[^\"]*"`), LITSTRING}, // double-quoted strings. escapes not supported
	{regexp.MustCompile(`^filetype\b`), FILETYPE},
	{regexp.MustCompile(`^stage\b`), STAGE},
	{regexp.MustCompile(`^pipeline\b`), PIPELINE},
	{regexp.MustCompile(`^call\b`), CALL},
	{regexp.MustCompile(`^` + local + `\b`), LOCAL},
	{regexp.MustCompile(`^` + preflight + `\b`), PREFLIGHT},
	{regexp.MustCompile(`^` + volatile + `\b`), VOLATILE},
	{regexp.MustCompile(`^` + disabled + `\b`), DISABLED},
	{regexp.MustCompile(`^` + strict + `\b`), STRICT},
	{regexp.MustCompile(`^threads\b`), THREADS},
	{regexp.MustCompile(`^mem_?gb\b`), MEM_GB},
	{regexp.MustCompile(`^vmem_?gb\b`), VMEM_GB},
	{regexp.MustCompile(`^special\b`), SPECIAL},
	{regexp.MustCompile(`^retain\b`), RETAIN},
	{regexp.MustCompile(`^sweep\b`), SWEEP},
	{regexp.MustCompile(`^split\b`), SPLIT},
	{regexp.MustCompile(`^using\b`), USING},
	{regexp.MustCompile(`^self\b`), SELF},
	{regexp.MustCompile(`^return\b`), RETURN},
	{regexp.MustCompile(`^in\b`), IN},
	{regexp.MustCompile(`^out\b`), OUT},
	{regexp.MustCompile(`^src\b`), SRC},
	{regexp.MustCompile(`^as\b`), AS},
	{regexp.MustCompile(`^` + abr_python + `\b`), PY},
	{regexp.MustCompile(`^` + abr_exec + `\b`), EXEC},
	{regexp.MustCompile(`^` + abr_compiled + `\b`), COMPILED},
	{regexp.MustCompile(`^map\b`), MAP},
	{regexp.MustCompile(`^int\b`), INT},
	{regexp.MustCompile(`^string\b`), STRING},
	{regexp.MustCompile(`^float\b`), FLOAT},
	{regexp.MustCompile(`^path\b`), PATH},
	{regexp.MustCompile(`^bool\b`), BOOL},
	{regexp.MustCompile(`^true\b`), TRUE},
	{regexp.MustCompile(`^false\b`), FALSE},
	{regexp.MustCompile(`^null\b`), NULL},
	{regexp.MustCompile(`^` + default_out_name + `\b`), DEFAULT},
	{regexp.MustCompile(`^_?[a-zA-Z][a-zA-z0-9_]*\b`), ID},
	{regexp.MustCompile(`^-?[0-9]+(:?\.[0-9]+[eE][+-]?|[eE][+-]?|\.)[0-9]+\b`), NUM_FLOAT},
	{regexp.MustCompile(`^-?0*?[0-9]{1,19}\b`), NUM_INT},
}

func nextToken(head []byte) (int, []byte) {
	for i := range rules {
		val := rules[i].re.Find(head)
		if len(val) > 0 {
			// Advance the cursor pos.
			return rules[i].tokid, val
		}
	}
	return INVALID, nil
}
