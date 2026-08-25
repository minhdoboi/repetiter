package providers

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type MockText struct{}

func (MockText) Complete(_ context.Context, req TextRequest) (Suggestion, error) {
	src := strings.TrimSpace(req.SourceText)
	if req.TargetLang == "vi" {
		return Suggestion{
			Translation:    fmt.Sprintf("%s (vi)", src),
			Reformulations: []string{src + " — cách khác", src + " — lịch sự hơn", src + " — ngắn gọn"},
			Related:        []string{"Có xa không?", "Đi bộ được không?", "Nên bắt xe buýt nào?"},
		}, nil
	}
	return Suggestion{
		Translation:    fmt.Sprintf("%s (fr)", src),
		Reformulations: []string{src + " — autrement", src + " — plus poli", src + " — plus court"},
		Related:        []string{"C’est loin ?", "On peut y aller à pied ?", "Quel bus faut-il prendre ?"},
	}, nil
}

type MockTTS struct{}

func (MockTTS) Synthesize(_ context.Context, _ TTSRequest) (io.ReadCloser, string, error) {
	return nil, "", fmt.Errorf("mock tts: use on-device speech")
}
