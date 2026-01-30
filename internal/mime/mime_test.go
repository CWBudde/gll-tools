package mime

import "testing"

func TestGuessMimeType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Supported extensions
		{"pdf", "document.pdf", "application/pdf"},
		{"png", "image.png", "image/png"},
		{"jpg", "photo.jpg", "image/jpeg"},
		{"jpeg", "photo.jpeg", "image/jpeg"},
		{"gif", "animation.gif", "image/gif"},
		{"xed", "geometry.xed", "application/octet-stream"},

		// Case insensitivity
		{"uppercase PDF", "FILE.PDF", "application/pdf"},
		{"mixed case Png", "Image.Png", "image/png"},
		{"uppercase JPG", "PHOTO.JPG", "image/jpeg"},

		// Unknown extensions
		{"unknown txt", "file.txt", "application/octet-stream"},
		{"unknown bmp", "image.bmp", "application/octet-stream"},

		// Edge cases
		{"empty string", "", "application/octet-stream"},
		{"no extension", "filename", "application/octet-stream"},
		{"dot only", ".", "application/octet-stream"},

		// Paths with separators
		{"unix path png", "/home/user/images/photo.png", "image/png"},
		{"windows path pdf", `C:\docs\file.pdf`, "application/pdf"},
		{"nested path jpg", "a/b/c/d.jpg", "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GuessMimeType(tt.input)
			if got != tt.expected {
				t.Errorf("GuessMimeType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
