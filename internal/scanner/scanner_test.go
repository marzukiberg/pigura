package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRootAllowsOnlyChildOfBase(t *testing.T) {
	base := t.TempDir()
	library := filepath.Join(base, "immich", "library")
	if err := os.MkdirAll(library, 0o755); err != nil {
		t.Fatal(err)
	}
	library, err := filepath.EvalSymlinks(library)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ValidateRoot(library, base)
	if err != nil {
		t.Fatalf("ValidateRoot() error = %v", err)
	}
	if got != library {
		t.Fatalf("ValidateRoot() = %q, want %q", got, library)
	}
	if _, err := ValidateRoot(base, base); err == nil {
		t.Fatal("ValidateRoot() allowed the scan base itself")
	}
	if _, err := ValidateRoot(filepath.Dir(base), base); err == nil {
		t.Fatal("ValidateRoot() allowed parent of scan base")
	}
}

func TestFindImmichLibrary(t *testing.T) {
	base := t.TempDir()
	want := filepath.Join(base, "immich", "library")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := FindImmichLibrary(base)
	if err != nil {
		t.Fatalf("FindImmichLibrary() error = %v", err)
	}
	if got != want {
		t.Fatalf("FindImmichLibrary() = %q, want %q", got, want)
	}
}
