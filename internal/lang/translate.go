package lang

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Translator interface {
	Translate(ctx context.Context, text, from, to string) (string, error)
	Name() string
	Version() string
}

type TranslationMeta struct {
	FromLanguage string
	ToLanguage   string
	Model        string
	ModelVersion string
	SourceHash   string
	OutputHash   string
}

var ErrNotConfigured = fmt.Errorf("lang: no translation provider configured")

type noopTranslator struct{}

func (noopTranslator) Translate(context.Context, string, string, string) (string, error) {
	return "", ErrNotConfigured
}
func (noopTranslator) Name() string    { return "none" }
func (noopTranslator) Version() string { return "0" }

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
