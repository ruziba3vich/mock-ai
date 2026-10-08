package application

import (
	"context"
	"io"
	"strings"
	"testing"
)

type importStub struct{}

func (importStub) UploadPDF(context.Context, string, io.Reader, int64) (string, string, error) {
	return "uri", "digest", nil
}
func (importStub) Extract(context.Context, string, string, string) ([]Content, []string, error) {
	return nil, nil, nil
}

func TestCreateImportRejectsUnsafeInputBeforeUpload(t *testing.T) {
	service := &Service{importer: importStub{}}
	tests := []struct {
		name, filename, body, skills string
		rights                       bool
		size                         int64
	}{
		{"rights required", "book.pdf", "%PDF-x", "all", false, 6},
		{"extension required", "book.txt", "%PDF-x", "all", true, 6},
		{"signature required", "book.pdf", "notpdf", "all", true, 6},
		{"known skill required", "book.pdf", "%PDF-x", "writing", true, 6},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.CreateImport(context.Background(), "admin", test.filename, test.skills, "public", "", test.rights, test.size, strings.NewReader(test.body)); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeImportedTestAddsMockDefaults(t *testing.T) {
	c := Content{Title: " Test 1 ", Skill: "reading", Sections: []ContentSection{{Title: " Passage ", Questions: make([]ContentQuestion, 40)}}}
	for i := range c.Sections[0].Questions {
		c.Sections[0].Questions[i] = ContentQuestion{Type: "short_answer", Prompt: " Q ", Answers: []string{" answer ", "answer"}, Explanation: " E "}
	}
	normalizeImported(&c)
	if c.Purpose != "mock" || c.Duration != 3600 || len(c.Bands) == 0 || c.Title != "Test 1" {
		t.Fatalf("defaults not applied: %+v", c)
	}
	if got := c.Sections[0].Questions[0].Answers; len(got) != 1 || got[0] != "answer" {
		t.Fatalf("answers not normalized: %v", got)
	}
}
