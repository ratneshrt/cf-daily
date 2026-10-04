package service

import "strings"

// defaultLanguage is used when a submission cannot be classified. It matches
// the extension every solution used before language detection existed, so
// existing files in the repository keep the same path.
const defaultLanguage = "cpp"

// detectLanguage guesses the language of a submission from its source, so the
// file lands in the repository with a sensible extension. It is deliberately
// conservative: anything unrecognised is treated as C++.
func detectLanguage(code string) string {
	lower := strings.ToLower(code)

	switch {
	case strings.Contains(lower, "#include") ||
		strings.Contains(lower, "std::") ||
		strings.Contains(lower, "cin >>") ||
		strings.Contains(lower, "cout <<"):
		return "cpp"

	case strings.Contains(lower, "import java.") ||
		strings.Contains(code, "public static void main"):
		return "java"

	case strings.Contains(lower, "package main") ||
		strings.Contains(code, "func main()"):
		return "go"

	case strings.Contains(code, "fn main()") ||
		strings.Contains(lower, "let mut "):
		return "rust"

	case strings.Contains(lower, "console.log") ||
		strings.Contains(lower, "require(") ||
		strings.Contains(lower, "readline()"):
		return "javascript"

	case strings.Contains(lower, "import sys") ||
		strings.Contains(lower, "input()") ||
		strings.Contains(lower, "print(") ||
		strings.Contains(code, "def "):
		return "python"

	default:
		return defaultLanguage
	}
}

// languageExtension maps a detected language to its file extension.
func languageExtension(language string) string {
	switch language {
	case "python":
		return "py"
	case "java":
		return "java"
	case "go":
		return "go"
	case "rust":
		return "rs"
	case "javascript":
		return "js"
	default:
		return "cpp"
	}
}

// submissionLanguage reads the language recorded with a submission, falling
// back to the default for rows written before languages were stored.
func submissionLanguage(language *string) string {
	if language == nil || *language == "" {
		return defaultLanguage
	}

	return *language
}
