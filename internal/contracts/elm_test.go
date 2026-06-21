package contracts

import (
	"strings"
	"testing"
)

func TestGenerateElmFilesIncludesAuthContract(t *testing.T) {
	result := GenerateElmFiles(Modules())
	generated, matched := result.(ElmFilesGenerated)
	if !matched {
		t.Fatalf("result = %T, want ElmFilesGenerated", result)
	}

	file := findElmFile(t, generated.Files, "web/elm/src/Sharecrop/Generated/Auth.elm")
	if !strings.Contains(file.Content, "type SubjectKind") {
		t.Fatalf("auth contract did not include SubjectKind")
	}
	if !strings.Contains(file.Content, "authResponseDecoder : Decoder AuthResponse") {
		t.Fatalf("auth contract did not include AuthResponse decoder")
	}
}

func TestGenerateElmFilesIsDeterministic(t *testing.T) {
	firstResult := GenerateElmFiles(Modules())
	secondResult := GenerateElmFiles(Modules())
	firstGenerated := firstResult.(ElmFilesGenerated)
	secondGenerated := secondResult.(ElmFilesGenerated)

	if len(firstGenerated.Files) != len(secondGenerated.Files) {
		t.Fatalf("file count = %d, want %d", len(firstGenerated.Files), len(secondGenerated.Files))
	}

	for index, first := range firstGenerated.Files {
		second := secondGenerated.Files[index]
		if first.Path != second.Path {
			t.Fatalf("path = %q, want %q", first.Path, second.Path)
		}
		if first.Content != second.Content {
			t.Fatalf("content for %s changed between generations", first.Path)
		}
	}
}

func TestGeneratedContractsAvoidWeakShapes(t *testing.T) {
	result := GenerateElmFiles(Modules())
	generated := result.(ElmFilesGenerated)

	for _, file := range generated.Files {
		if strings.Contains(file.Content, "Bool") {
			t.Fatalf("%s contains Bool", file.Path)
		}
		if strings.Contains(file.Content, "Dict") {
			t.Fatalf("%s contains Dict", file.Path)
		}
	}
}

func findElmFile(t *testing.T, files []ElmFile, path string) ElmFile {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("generated file %s was not found", path)
	return ElmFile{}
}
