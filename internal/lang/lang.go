package lang

import (
	"context"
	"log/slog"
)

type Service struct {
	detector   *Detector
	translator Translator
	target     string
}

type Result struct {
	Language       string
	Confidence     float64
	TranslatedBody string
	Translation    *TranslationMeta
}

func NewService(d *Detector, t Translator, target string) *Service {
	if t == nil {
		t = noopTranslator{}
	}
	return &Service{detector: d, translator: t, target: target}
}

func (s *Service) Process(ctx context.Context, title, body, hint string) Result {
	text := title + "\n\n" + body
	detected, conf := s.detector.Detect(text)

	final := detected
	if conf < MinConfidence {
		if hint != "" && hint != "und" {
			final = hint
		}
	}

	res := Result{Language: final, Confidence: conf}
	if final == "und" || final == s.target {
		return res
	}

	out, err := s.translator.Translate(ctx, body, final, s.target)
	if err != nil {
		if err != ErrNotConfigured {
			slog.Warn("translate failed", "from", final, "to", s.target, "err", err)
		}
		return res
	}
	if out == "" {
		return res
	}

	res.TranslatedBody = out
	res.Translation = &TranslationMeta{
		FromLanguage: final,
		ToLanguage:   s.target,
		Model:        s.translator.Name(),
		ModelVersion: s.translator.Version(),
		SourceHash:   hashString(body),
		OutputHash:   hashString(out),
	}
	return res
}
