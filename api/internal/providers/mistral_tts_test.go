package providers

import (
	"encoding/base64"
	"testing"
)

func TestDecodeMistralSpeechBodyJSON(t *testing.T) {
	mp3 := []byte{0xff, 0xfb, 0x90, 0x00}
	encoded := base64.StdEncoding.EncodeToString(mp3)
	raw := []byte(`{"audio_data":"` + encoded + `"}`)
	audio, ct, err := decodeMistralSpeechBody(raw, "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if string(audio) != string(mp3) {
		t.Fatalf("audio = %v, want %v", audio, mp3)
	}
	if ct != "audio/mpeg" {
		t.Fatalf("content type = %q", ct)
	}
}

func TestDecodeMistralSpeechBodyBinary(t *testing.T) {
	mp3 := []byte{0xff, 0xfb, 0x90, 0x00}
	audio, ct, err := decodeMistralSpeechBody(mp3, "audio/mpeg")
	if err != nil {
		t.Fatal(err)
	}
	if string(audio) != string(mp3) {
		t.Fatal("binary passthrough failed")
	}
	if ct != "audio/mpeg" {
		t.Fatalf("content type = %q", ct)
	}
}

func TestSniffAudioContentType(t *testing.T) {
	wav := []byte("RIFF....WAVE")
	if sniffAudioContentType(wav) != "audio/wav" {
		t.Fatal("expected wav")
	}
}
