package application

import "testing"

func TestPublishedContentKindsIncludeStudentAssessmentLibraries(t *testing.T) {
	for _, kind := range []string{"writing_prompt", "speaking_plan"} {
		if !isPublishedContentKind(kind) {
			t.Fatalf("expected %q to be available from the published content catalog", kind)
		}
	}

	if isPublishedContentKind("admin_only") {
		t.Fatal("unexpected unsupported content kind in the published catalog")
	}
}
