package version

import "testing"

func TestShowReturnsInjectedVersion(t *testing.T) {
	t.Parallel()
	uc := NewUseCase("1.2.3")
	if got := uc.Show(); got.Version != "1.2.3" {
		t.Fatalf("Show().Version = %q, want %q", got.Version, "1.2.3")
	}
}

func TestShowDefaultsToDev(t *testing.T) {
	t.Parallel()
	uc := NewUseCase("")
	if got := uc.Show(); got.Version != "dev" {
		t.Fatalf("Show().Version = %q, want %q", got.Version, "dev")
	}
}
