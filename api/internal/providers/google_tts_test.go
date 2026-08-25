package providers

import (
	"testing"

	"repetiter/internal/config"
)

func TestChunkGoogleTTS(t *testing.T) {
	short := "Bonjour le monde"
	got := chunkGoogleTTS(short, 200)
	if len(got) != 1 || got[0] != short {
		t.Fatalf("short text: got %v", got)
	}

	long := ""
	for i := 0; i < 250; i++ {
		long += "a"
	}
	got = chunkGoogleTTS(long, 200)
	if len(got) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(got))
	}
	for i, c := range got {
		if len([]rune(c)) > 200 {
			t.Fatalf("chunk %d too long: %d runes", i, len([]rune(c)))
		}
	}

	punct := "Première phrase. Deuxième phrase plus longue pour tester la coupure."
	got = chunkGoogleTTS(punct, 20)
	if len(got) < 2 {
		t.Fatalf("expected split on punctuation, got %v", got)
	}
}

func TestValidGoogleTTSVoice(t *testing.T) {
	if !ValidTTSVoice("google", "normal") {
		t.Fatal("normal should be valid")
	}
	if !ValidTTSVoice("google", "slow") {
		t.Fatal("slow should be valid")
	}
	if ValidTTSVoice("google", "alloy") {
		t.Fatal("alloy should not be valid for google")
	}
}

func TestDefaultGoogleTTSVoice(t *testing.T) {
	got := DefaultTTSVoice("google", config.Config{})
	if got != "normal" {
		t.Fatalf("DefaultTTSVoice = %q, want normal", got)
	}
}
