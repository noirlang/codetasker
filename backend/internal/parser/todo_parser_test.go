package parser

import (
	"reflect"
	"testing"
)

func TestParseFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []ParsedTask
	}{
		{
			name: "single line comments",
			content: `
// TODO: implement single line
# FIXME: fix this bug
`,
			want: []ParsedTask{
				{FilePath: "test.go", LineNumber: 2, Type: "TODO", Content: "implement single line"},
				{FilePath: "test.go", LineNumber: 3, Type: "FIXME", Content: "fix this bug"},
			},
		},
		{
			name: "multi-line comments with list items",
			content: `
// TODO: parent task
//   - subtask 1
//   - subtask 2
// and this should not merge because it's not indented or list
`,
			want: []ParsedTask{
				{
					FilePath:   "test.go",
					LineNumber: 2,
					Type:       "TODO",
					Content:    "parent task\n- subtask 1\n- subtask 2",
				},
			},
		},
		{
			name: "block comments multi-line",
			content: `
/*
 * FIXME: parent block
 *   - bullet 1
 *   - bullet 2
 */
`,
			want: []ParsedTask{
				{
					FilePath:   "test.go",
					LineNumber: 3,
					Type:       "FIXME",
					Content:    "parent block\n- bullet 1\n- bullet 2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := FileContent{
				Path:    "test.go",
				Content: tt.content,
			}
			got := ParseFile(fc)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
