package debt

import (
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	todoPattern      = regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|XXX)\b`)
	goFunction       = regexp.MustCompile(`^\s*func\s+`)
	pythonFunction   = regexp.MustCompile(`^\s*(async\s+)?def\s+`)
	jsFunction       = regexp.MustCompile(`\bfunction\b|=>|^\s*(public|private|protected|static|async|\s)*[A-Za-z_$][\w$]*\s*\([^)]*\)\s*\{`)
	importLine       = regexp.MustCompile(`^\s*(import\s+.+|from\s+\S+\s+import\s+.+)$`)
	goImportSingle   = regexp.MustCompile(`^\s*import\s+"([^"]+)"`)
	goImportBlockPkg = regexp.MustCompile(`^\s*(?:[._A-Za-z0-9]+\s+)?"([^"]+)"`)
)

func SupportedPath(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".py", ".go":
		return true
	default:
		return false
	}
}

func IsIgnoredPath(filePath string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(filePath, "\\", "/"))
	parts := strings.Split(normalized, "/")
	ignoredDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		"target":       true,
		".next":        true,
		".nuxt":        true,
		"coverage":     true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
	}
	for _, part := range parts {
		if ignoredDirs[part] {
			return true
		}
	}
	return false
}

func IsTestPath(filePath string) bool {
	base := strings.ToLower(path.Base(strings.ReplaceAll(filePath, "\\", "/")))
	dir := strings.ToLower(path.Dir(strings.ReplaceAll(filePath, "\\", "/")))
	return strings.Contains(dir, "__tests__") ||
		strings.Contains(dir, "/test") ||
		strings.Contains(dir, "/tests") ||
		strings.HasPrefix(base, "test_") ||
		strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") ||
		strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.py")
}

func AnalyzeStatic(files []SourceFile, allPaths []string) map[string]Metrics {
	pathSet := make(map[string]struct{}, len(allPaths))
	for _, p := range allPaths {
		pathSet[strings.ReplaceAll(p, "\\", "/")] = struct{}{}
	}

	result := make(map[string]Metrics, len(files))
	for _, file := range files {
		if !SupportedPath(file.Path) || IsIgnoredPath(file.Path) || IsTestPath(file.Path) {
			continue
		}

		metrics := analyzeFile(file.Path, file.Content)
		metrics.HasTests = hasMatchingTest(file.Path, pathSet)
		if metrics.HasTests {
			metrics.CoverageStatus = "test_file_detected"
		} else {
			metrics.CoverageStatus = "not_detected"
		}
		result[file.Path] = metrics
	}
	return result
}

func analyzeFile(filePath, content string) Metrics {
	lines := strings.Split(content, "\n")
	ext := strings.ToLower(filepath.Ext(filePath))

	metrics := Metrics{
		LOC:                  countLOC(lines),
		TodoCount:            len(todoPattern.FindAllString(content, -1)),
		DuplicateImportCount: countDuplicateImports(lines, ext),
	}

	functionLengths := estimateFunctionLengths(lines, ext)
	metrics.FunctionCount = len(functionLengths)
	if len(functionLengths) > 0 {
		total := 0
		for _, length := range functionLengths {
			total += length
			if length > metrics.MaxFunctionLength {
				metrics.MaxFunctionLength = length
			}
		}
		metrics.AvgFunctionLength = float64(total) / float64(len(functionLengths))
	}

	metrics.NestingDepthEstimate = estimateNestingDepth(lines, ext)
	metrics.CyclomaticComplexityEstimate = estimateCyclomaticComplexity(content, ext, metrics.FunctionCount)
	return metrics
}

func countLOC(lines []string) int {
	loc := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		loc++
	}
	return loc
}

func countDuplicateImports(lines []string, ext string) int {
	imports := make(map[string]int)
	inGoBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		switch ext {
		case ".go":
			if strings.HasPrefix(trimmed, "import (") {
				inGoBlock = true
				continue
			}
			if inGoBlock {
				if trimmed == ")" {
					inGoBlock = false
					continue
				}
				if match := goImportBlockPkg.FindStringSubmatch(trimmed); len(match) == 2 {
					imports[match[1]]++
				}
				continue
			}
			if match := goImportSingle.FindStringSubmatch(trimmed); len(match) == 2 {
				imports[match[1]]++
			}
		default:
			if importLine.MatchString(trimmed) {
				imports[trimmed]++
			}
		}
	}

	duplicates := 0
	for _, count := range imports {
		if count > 1 {
			duplicates += count - 1
		}
	}
	return duplicates
}

func estimateFunctionLengths(lines []string, ext string) []int {
	switch ext {
	case ".py":
		return estimatePythonFunctionLengths(lines)
	case ".go":
		return estimateBraceFunctionLengths(lines, goFunction)
	default:
		return estimateBraceFunctionLengths(lines, jsFunction)
	}
}

func estimatePythonFunctionLengths(lines []string) []int {
	var lengths []int
	for i := 0; i < len(lines); i++ {
		if !pythonFunction.MatchString(lines[i]) {
			continue
		}
		startIndent := leadingSpaces(lines[i])
		end := i + 1
		for end < len(lines) {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed != "" && leadingSpaces(lines[end]) <= startIndent && (pythonFunction.MatchString(lines[end]) || strings.HasPrefix(trimmed, "class ")) {
				break
			}
			end++
		}
		lengths = append(lengths, end-i)
	}
	return lengths
}

func estimateBraceFunctionLengths(lines []string, startPattern *regexp.Regexp) []int {
	var lengths []int
	for i := 0; i < len(lines); i++ {
		if !startPattern.MatchString(lines[i]) {
			continue
		}

		depth := 0
		seenOpen := false
		end := i
		for ; end < len(lines); end++ {
			for _, r := range stripInlineComment(lines[end]) {
				switch r {
				case '{':
					depth++
					seenOpen = true
				case '}':
					if depth > 0 {
						depth--
					}
				}
			}
			if seenOpen && depth == 0 {
				break
			}
		}
		if end < i {
			end = i
		}
		lengths = append(lengths, end-i+1)
	}
	return lengths
}

func estimateNestingDepth(lines []string, ext string) int {
	if ext == ".py" {
		maxDepth := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			depth := leadingSpaces(line) / 4
			if depth > maxDepth {
				maxDepth = depth
			}
		}
		return maxDepth
	}

	maxDepth := 0
	depth := 0
	for _, line := range lines {
		for _, r := range stripInlineComment(line) {
			switch r {
			case '{':
				depth++
				if depth > maxDepth {
					maxDepth = depth
				}
			case '}':
				if depth > 0 {
					depth--
				}
			}
		}
	}
	return maxDepth
}

func estimateCyclomaticComplexity(content, ext string, functionCount int) int {
	patterns := []string{`\bif\b`, `\bfor\b`, `\bwhile\b`, `\bcase\b`, `\bcatch\b`, `&&`, `\|\|`, `\?`}
	if ext == ".py" {
		patterns = []string{`\bif\b`, `\belif\b`, `\bfor\b`, `\bwhile\b`, `\bexcept\b`, `\band\b`, `\bor\b`, `\bcase\b`}
	}
	if ext == ".go" {
		patterns = []string{`\bif\b`, `\bfor\b`, `\bcase\b`, `\bswitch\b`, `\bselect\b`, `&&`, `\|\|`}
	}

	complexity := 1
	if functionCount > 0 {
		complexity = functionCount
	}
	for _, p := range patterns {
		complexity += len(regexp.MustCompile(p).FindAllString(content, -1))
	}
	return complexity
}

func hasMatchingTest(filePath string, pathSet map[string]struct{}) bool {
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	if IsTestPath(normalized) {
		return true
	}

	dir := path.Dir(normalized)
	base := path.Base(normalized)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)

	candidates := []string{}
	switch ext {
	case ".go":
		candidates = append(candidates, path.Join(dir, name+"_test.go"))
	case ".py":
		candidates = append(candidates,
			path.Join(dir, "test_"+name+".py"),
			path.Join(dir, name+"_test.py"),
			path.Join("tests", dir, "test_"+name+".py"),
		)
	case ".ts", ".tsx", ".js", ".jsx":
		candidates = append(candidates,
			path.Join(dir, name+".test"+ext),
			path.Join(dir, name+".spec"+ext),
			path.Join(dir, "__tests__", base),
		)
	}

	sort.Strings(candidates)
	for _, candidate := range candidates {
		if _, ok := pathSet[candidate]; ok {
			return true
		}
	}
	return false
}

func leadingSpaces(line string) int {
	count := 0
	for _, r := range line {
		switch r {
		case ' ':
			count++
		case '\t':
			count += 4
		default:
			return count
		}
	}
	return count
}

func stripInlineComment(line string) string {
	if idx := strings.Index(line, "//"); idx >= 0 {
		return line[:idx]
	}
	if idx := strings.Index(line, "#"); idx >= 0 {
		return line[:idx]
	}
	return line
}
