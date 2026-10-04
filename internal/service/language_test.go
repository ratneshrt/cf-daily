package service

import "testing"

func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		name string
		code string
		want string
	}{
		{
			name: "cpp",
			code: "#include <bits/stdc++.h>\nint main(){int n;std::cin>>n;}",
			want: "cpp",
		},
		{
			name: "python",
			code: "import sys\nn = int(input())\nprint(n)",
			want: "python",
		},
		{
			name: "java",
			code: "import java.util.*;\npublic class Main { public static void main(String[] a){} }",
			want: "java",
		},
		{
			name: "go",
			code: "package main\n\nfunc main() {}",
			want: "go",
		},
		{
			name: "rust",
			code: "fn main() { let mut n = 0; }",
			want: "rust",
		},
		{
			name: "javascript",
			code: "const n = readline()\nconsole.log(n)",
			want: "javascript",
		},
		{
			name: "unknown falls back to cpp",
			code: "some notes that are not code at all",
			want: defaultLanguage,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := detectLanguage(testCase.code); got != testCase.want {
				t.Fatalf("detectLanguage() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestLanguageExtension(t *testing.T) {
	cases := map[string]string{
		"cpp":        "cpp",
		"python":     "py",
		"java":       "java",
		"go":         "go",
		"rust":       "rs",
		"javascript": "js",
		"something":  "cpp",
	}

	for language, want := range cases {
		if got := languageExtension(language); got != want {
			t.Errorf("languageExtension(%q) = %q, want %q", language, got, want)
		}
	}
}

func TestSubmissionLanguage(t *testing.T) {
	if got := submissionLanguage(nil); got != defaultLanguage {
		t.Errorf("nil language = %q, want %q", got, defaultLanguage)
	}

	empty := ""

	if got := submissionLanguage(&empty); got != defaultLanguage {
		t.Errorf("empty language = %q, want %q", got, defaultLanguage)
	}

	python := "python"

	if got := submissionLanguage(&python); got != "python" {
		t.Errorf("stored language = %q, want python", got)
	}
}

func TestBuildSolutionPath(t *testing.T) {
	cases := []struct {
		language string
		want     string
	}{
		{language: "cpp", want: "1956/1956D-Nene-and-the-Passing-Game/solution.cpp"},
		{language: "python", want: "1956/1956D-Nene-and-the-Passing-Game/solution.py"},
	}

	for _, testCase := range cases {
		got := buildSolutionPath(1956, "D", "Nene and the Passing Game", testCase.language)

		if got != testCase.want {
			t.Errorf("buildSolutionPath(%q) = %q, want %q", testCase.language, got, testCase.want)
		}
	}
}

func TestSanitizerProblemName(t *testing.T) {
	cases := map[string]string{
		"Watermelon":            "Watermelon",
		"A + B":                 "A-B",
		"  Nene's Magical  Box": "Nene-s-Magical-Box",
		"***":                   "",
	}

	for input, want := range cases {
		if got := sanitizerProblemName(input); got != want {
			t.Errorf("sanitizerProblemName(%q) = %q, want %q", input, got, want)
		}
	}
}
