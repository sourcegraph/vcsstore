package vcsclient

import "testing"

func TestComputeFileRange(t *testing.T) {
	tests := map[string]struct {
		data []byte
		opt  GetFileOptions
		want FileRange
	}{
		"zero": {
			data: []byte(``),
			opt:  GetFileOptions{},
			want: FileRange{StartLine: 0, EndLine: 0},
		},
		"1 char": {
			data: []byte(`a`),
			opt:  GetFileOptions{},
			want: FileRange{StartLine: 1, EndLine: 1, EndByte: 1},
		},
		"1 line": {
			data: []byte("a\n"),
			opt:  GetFileOptions{},
			want: FileRange{StartLine: 1, EndLine: 1, EndByte: 2},
		},
		"2 lines, no trailing newline": {
			data: []byte("a\nb"),
			opt:  GetFileOptions{},
			want: FileRange{StartLine: 1, EndLine: 2, EndByte: 3},
		},
		"2 lines, trailing newline": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{},
			want: FileRange{StartLine: 1, EndLine: 2, EndByte: 4},
		},
		"2 lines, byte range": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartByte: 2, EndByte: 3}},
			want: FileRange{StartLine: 2, EndLine: 2, StartByte: 2, EndByte: 3},
		},
		"2 lines, byte range, full lines": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartByte: 2, EndByte: 3}, FullLines: true},
			want: FileRange{StartLine: 2, EndLine: 2, StartByte: 2, EndByte: 4},
		},
		"2 lines, line range": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartLine: 2, EndLine: 2}},
			want: FileRange{StartLine: 2, EndLine: 2, StartByte: 2, EndByte: 4},
		},
		"OOB end line": {
			data: []byte("a\nb"),
			opt:  GetFileOptions{FileRange: FileRange{StartLine: 0, EndLine: 999999}},
			want: FileRange{StartLine: 2, EndLine: 2, StartByte: 2, EndByte: 4},
		},
	}
	for label, test := range tests {
		got, _, err := ComputeFileRange(test.data, test.opt)
		if err != nil {
			t.Errorf("%s: ComputeFileRange error: %s", label, err)
			continue
		}

		if *got != test.want {
			t.Errorf("%s: got %+v, want %+v", label, *got, test.want)
		}

		// Validate indices against data.
		_ = test.data[got.StartByte:got.EndByte]
	}
}

func TestComputeBadFileRange(t *testing.T) {
	tests := map[string]struct {
		data []byte
		opt  GetFileOptions
		want string
	}{
		// Byte ranges
		"negative start byte": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartByte: -1, EndByte: 3}},
			want: "start byte -1 out of bounds (4 bytes total)",
		},
		"OOB end byte": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartByte: 0, EndByte: 5}},
			want: "end byte 5 out of bounds (4 bytes total)",
		},
		"inverse valid bytes": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartByte: 3, EndByte: 0}},
			want: "start byte (3) cannot be greater than end byte (0) (4 bytes total)",
		},
		"inverse 2 lines, byte range": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartByte: 3, EndByte: 2}},
			want: "start byte (3) cannot be greater than end byte (2) (4 bytes total)",
		},
		"inverse 2 lines, byte range, full lines": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartByte: 3, EndByte: 2}, FullLines: true},
			want: "start byte (3) cannot be greater than end byte (2) (4 bytes total)",
		},

		// Line ranges
		"negative start line": {
			data: []byte("a\nb"),
			opt:  GetFileOptions{FileRange: FileRange{StartLine: -1, EndLine: 1}},
			want: "start line -1 out of bounds (2 lines total)",
		},
		"inverse 2 lines, no trailing newline": {
			data: []byte("a\nb"),
			opt:  GetFileOptions{FileRange: FileRange{StartLine: 2, EndLine: 1}},
			want: "start line (2) cannot be greater than end line (1) (2 lines total)",
		},
		"inverse 2 lines, trailing newline": {
			data: []byte("a\nb\n"),
			opt:  GetFileOptions{FileRange: FileRange{StartLine: 2, EndLine: 1}},
			want: "start line (2) cannot be greater than end line (1) (2 lines total)",
		},
	}
	for label, test := range tests {
		got, _, err := ComputeFileRange(test.data, test.opt)
		if err == nil || err.Error() != test.want {
			t.Errorf("%s: got %q, want %q", label, err, test.want)
			continue
		}
		if got != nil {
			t.Errorf("%s: expected nil result", label)
		}
	}
}
