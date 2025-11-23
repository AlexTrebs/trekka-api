package utils

import (
	"testing"
)

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "ISO 8601 UTC format",
			input:     "2025-01-15T14:30:00Z",
			wantError: false,
		},
		{
			name:      "EXIF format",
			input:     "2025:01:15 14:30:00",
			wantError: false,
		},
		{
			name:      "ISO format without Z",
			input:     "2025-01-15T14:30:00",
			wantError: false,
		},
		{
			name:      "Christmas example",
			input:     "2024:12:25 09:15:00",
			wantError: false,
		},
		{
			name:      "Invalid format",
			input:     "not a timestamp",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatTimestamp(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("FormatTimestamp(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("FormatTimestamp(%q) unexpected error: %v", tt.input, err)
				}
				if result == "" {
					t.Errorf("FormatTimestamp(%q) returned empty string", tt.input)
				}
				t.Logf("FormatTimestamp(%q) = %q", tt.input, result)
			}
		})
	}
}
