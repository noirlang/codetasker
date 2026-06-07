// Package parser implements CodeTasker's core TODO extraction engine.
// It scans source file contents for annotated comments (TODO, FIXME, HACK,
// BUG, NOTE) across multiple comment styles (C-style //, Python/Shell #,
// SQL/Haskell --, and block /* */) and returns structured ParsedTask values.
//
// Concurrency design:
//
// ParseFiles uses a classic fan-out/fan-in worker pool pattern:
//
//   main goroutine → work channel → N worker goroutines → results channel → collector
//
// The number of workers defaults to runtime.NumCPU() when the caller passes 0,
// giving full CPU utilisation on multi-core machines. A sync.WaitGroup ensures
// all workers finish before the results channel is closed, preventing a race
// between the collector goroutine reading from the channel and the workers
// writing to it.
package parser

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// annotationPattern matches TODO/FIXME/HACK/BUG/NOTE annotations in common
// single-line comment styles:
//
//	//  TODO: message   (C, Go, Java, Rust, …)
//	#   TODO: message   (Python, Ruby, Shell, YAML, …)
//	--  TODO: message   (SQL, Haskell, Lua, …)
//
// The pattern is case-insensitive (?i) so "todo", "Todo", "TODO" all match.
// Capturing groups: [1] keyword, [2] message text.
var annotationPattern = regexp.MustCompile(
	`(?i)(?://|#|--)\s*(TODO|FIXME|HACK|BUG|NOTE)[:\s]+(.*?)\s*$`,
)

// blockCommentPattern matches TODO-style annotations inside block comments:
//
//	/* TODO: message */
//	* TODO: message       (inside a /* … */ block, line starting with *)
var blockCommentPattern = regexp.MustCompile(
	`(?i)/?\*+\s*(TODO|FIXME|HACK|BUG|NOTE)[:\s]+(.*?)\s*(?:\*/\s*)?$`,
)

// ParsedTask holds the extracted information from a single annotation line.
// All fields are populated by ParseFile and consumed by the task service to
// upsert Task documents in MongoDB.
type ParsedTask struct {
	// FilePath is the repository-relative path of the file (e.g. "cmd/main.go").
	FilePath string

	// LineNumber is the 1-based line number of the annotation within the file.
	LineNumber int

	// Type is the keyword found: TODO, FIXME, HACK, BUG, or NOTE.
	Type string

	// Content is the trimmed message text following the keyword.
	Content string
}

// FileContent pairs a file path with its decoded string content.
// GithubService populates these structs before handing them to ParseFiles.
type FileContent struct {
	// Path is the repository-relative file path.
	Path string

	// Content is the full UTF-8 decoded source text of the file.
	Content string
}

// ParseFile scans a single file's content for TODO-style annotations and
// returns one ParsedTask per matching line. The scan is line-by-line so the
// line number is always accurate, even for files with inconsistent newlines.
//
// Both single-line comment styles (//, #, --) and block comment openings
// (/* … */ or * … inside a block) are recognised.
func ParseFile(fc FileContent) []ParsedTask {
	var tasks []ParsedTask
	lines := strings.Split(fc.Content, "\n")

	for i, line := range lines {
		lineNum := i + 1 // convert to 1-based

		// Try single-line comment styles first (more common).
		if m := annotationPattern.FindStringSubmatch(line); m != nil {
			tasks = append(tasks, ParsedTask{
				FilePath:   fc.Path,
				LineNumber: lineNum,
				Type:       strings.ToUpper(m[1]),
				Content:    strings.TrimSpace(m[2]),
			})
			continue
		}

		// Fall back to block comment style.
		if m := blockCommentPattern.FindStringSubmatch(line); m != nil {
			tasks = append(tasks, ParsedTask{
				FilePath:   fc.Path,
				LineNumber: lineNum,
				Type:       strings.ToUpper(m[1]),
				Content:    strings.TrimSpace(m[2]),
			})
		}
	}

	return tasks
}

// ParseFiles processes a slice of FileContent concurrently using a worker pool
// of the requested size and returns the aggregated list of ParsedTask values.
//
// If workers ≤ 0, the pool size defaults to runtime.NumCPU() so that the
// caller does not need to tune the value for each deployment target.
//
// Concurrency model:
//  1. A producer goroutine enqueues FileContent values into a buffered work channel.
//  2. `workers` consumer goroutines each call ParseFile on received items and
//     send the resulting []ParsedTask slice into a results channel.
//  3. A WaitGroup is decremented by each worker when it returns. A separate
//     goroutine waits for the WaitGroup to reach zero before closing the results
//     channel, signalling the collector that no more data is coming.
//  4. The collector (main goroutine) ranges over the results channel and
//     accumulates all tasks into a single slice.
//
// This design means the caller blocks until all files have been processed,
// which matches the synchronous expectations of the webhook handler.
func ParseFiles(files []FileContent, workers int) []ParsedTask {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Buffer the work channel to the number of files so the producer never blocks.
	workCh := make(chan FileContent, len(files))

	// Buffer the results channel generously to avoid worker back-pressure.
	resultCh := make(chan []ParsedTask, workers*2)

	var wg sync.WaitGroup

	// ── Worker pool ─────────────────────────────────────────────────────────
	// Each worker pulls FileContent items from workCh until it is closed,
	// parses the file, and forwards results without locking.
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fc := range workCh {
				parsed := ParseFile(fc)
				if len(parsed) > 0 {
					resultCh <- parsed
				}
			}
		}()
	}

	// ── Producer ────────────────────────────────────────────────────────────
	// Enqueue all files; the buffered channel never blocks.
	for _, f := range files {
		workCh <- f
	}
	close(workCh) // Signal workers that no more items are coming.

	// ── Closer goroutine ────────────────────────────────────────────────────
	// Wait for all workers to finish, then close resultCh so the collector
	// loop below terminates cleanly.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// ── Collector ───────────────────────────────────────────────────────────
	// Accumulate results from all workers into a single slice. This runs in
	// the caller's goroutine, blocking until resultCh is closed.
	var all []ParsedTask
	for batch := range resultCh {
		all = append(all, batch...)
	}

	return all
}

// Parser is a stateless struct that exists so service constructors can receive
// the parser as a dependency and call its methods via an interface if needed.
// All parsing logic lives in the package-level functions above.
type Parser struct{}

// NewParser constructs a Parser value. It has no configuration; the compiled
// regexes are package-level variables initialised once at program start.
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile is a method wrapper around the package-level ParseFile function,
// allowing Parser to be used wherever an interface is expected.
func (p *Parser) ParseFile(fc FileContent) []ParsedTask {
	return ParseFile(fc)
}

// ParseFiles is a method wrapper around the package-level ParseFiles function.
func (p *Parser) ParseFiles(files []FileContent, workers int) []ParsedTask {
	return ParseFiles(files, workers)
}

// String implements the Stringer interface for ParsedTask, useful for debug logging.
func (t ParsedTask) String() string {
	return fmt.Sprintf("[%s] %s:%d — %s", t.Type, t.FilePath, t.LineNumber, t.Content)
}
