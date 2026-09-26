package lang

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pemistahl/lingua-go"
)

const MinConfidence = 0.7
const MinTextRunes = 20

type Detector struct {
	once     sync.Once
	detector lingua.LanguageDetector
	langs    []lingua.Language
}

func NewDetector(isoCodes []string) (*Detector, error) {
	if len(isoCodes) == 0 {
		return nil, fmt.Errorf("lang: no languages configured")
	}
	langs := make([]lingua.Language, 0, len(isoCodes))
	for _, code := range isoCodes {
		code = strings.ToLower(strings.TrimSpace(code))
		if code == "" {
			continue
		}
		l, ok := isoToLingua[code]
		if !ok {
			return nil, fmt.Errorf("lang: unsupported ISO code %q", code)
		}
		langs = append(langs, l)
	}
	if len(langs) == 0 {
		return nil, fmt.Errorf("lang: no valid languages configured")
	}
	return &Detector{langs: langs}, nil
}

func (d *Detector) Detect(text string) (string, float64) {
	if len([]rune(text)) < MinTextRunes {
		return "und", 0
	}
	d.once.Do(func() {
		d.detector = lingua.NewLanguageDetectorBuilder().
			FromLanguages(d.langs...).
			Build()
	})
	lang, ok := d.detector.DetectLanguageOf(text)
	if !ok {
		return "und", 0
	}
	conf := d.detector.ComputeLanguageConfidence(text, lang)
	return strings.ToLower(lang.IsoCode639_1().String()), conf
}

var isoToLingua = map[string]lingua.Language{
	"af": lingua.Afrikaans, "ar": lingua.Arabic, "az": lingua.Azerbaijani,
	"be": lingua.Belarusian, "bg": lingua.Bulgarian, "bn": lingua.Bengali,
	"bs": lingua.Bosnian, "ca": lingua.Catalan, "cs": lingua.Czech,
	"da": lingua.Danish, "de": lingua.German, "el": lingua.Greek,
	"en": lingua.English, "es": lingua.Spanish, "et": lingua.Estonian,
	"fa": lingua.Persian, "fi": lingua.Finnish, "fr": lingua.French,
	"he": lingua.Hebrew, "hi": lingua.Hindi, "hr": lingua.Croatian,
	"hu": lingua.Hungarian, "hy": lingua.Armenian, "id": lingua.Indonesian,
	"is": lingua.Icelandic, "it": lingua.Italian, "ja": lingua.Japanese,
	"ka": lingua.Georgian, "kk": lingua.Kazakh, "ko": lingua.Korean,
	"lt": lingua.Lithuanian, "lv": lingua.Latvian, "mk": lingua.Macedonian,
	"mn": lingua.Mongolian, "ms": lingua.Malay, "nl": lingua.Dutch,
	"no": lingua.Bokmal, "pl": lingua.Polish, "pt": lingua.Portuguese,
	"ro": lingua.Romanian, "ru": lingua.Russian, "sk": lingua.Slovak,
	"sl": lingua.Slovene, "so": lingua.Somali, "sq": lingua.Albanian,
	"sr": lingua.Serbian, "sv": lingua.Swedish, "sw": lingua.Swahili,
	"ta": lingua.Tamil, "te": lingua.Telugu, "th": lingua.Thai,
	"tl": lingua.Tagalog, "tr": lingua.Turkish, "uk": lingua.Ukrainian,
	"ur": lingua.Urdu, "vi": lingua.Vietnamese, "zh": lingua.Chinese,
	"zu": lingua.Zulu,
}
