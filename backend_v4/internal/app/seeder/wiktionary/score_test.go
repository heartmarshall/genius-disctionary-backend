package wiktionary

import (
	"math"
	"testing"
)

func TestScoreEntry(t *testing.T) {
	tests := []struct {
		name string
		entry kaikkiEntry
		want  float64
	}{
		{
			name:  "empty entry",
			entry: kaikkiEntry{Word: "test"},
			want:  1.0, // single-word bonus only
		},
		{
			name:  "completely empty entry with multi-word",
			entry: kaikkiEntry{Word: "give up on"},
			want:  0.0,
		},
		{
			name: "one sense with one gloss, single word",
			entry: kaikkiEntry{
				Word: "hello",
				Senses: []kaikkiSense{
					{Glosses: []string{"a greeting"}},
				},
			},
			want: 2.0, // 1.0 sense + 1.0 single-word
		},
		{
			name: "sense with empty glosses is not counted",
			entry: kaikkiEntry{
				Word: "hello",
				Senses: []kaikkiSense{
					{Glosses: []string{}},
					{Glosses: nil},
					{Glosses: []string{"a greeting"}},
				},
			},
			want: 2.0, // only 1 valid sense (1.0) + 1.0 single-word
		},
		{
			name: "russian translations add 0.5 each",
			entry: kaikkiEntry{
				Word: "cat",
				Senses: []kaikkiSense{
					{
						Glosses: []string{"a feline animal"},
						Translations: []kaikkiTranslation{
							{Code: "ru", Word: "кот"},
							{Code: "ru", Word: "кошка"},
							{Code: "de", Word: "Katze"},
						},
					},
				},
			},
			want: 3.0, // 1.0 sense + 0.5 + 0.5 ru translations + 1.0 single-word
		},
		{
			name: "non-russian translations only give no bonus",
			entry: kaikkiEntry{
				Word: "cat",
				Senses: []kaikkiSense{
					{
						Glosses: []string{"a feline animal"},
						Translations: []kaikkiTranslation{
							{Code: "de", Word: "Katze"},
							{Code: "fr", Word: "chat"},
						},
					},
				},
			},
			want: 2.0, // 1.0 sense + 1.0 single-word, no translation bonus
		},
		{
			name: "examples add 0.3 each",
			entry: kaikkiEntry{
				Word: "run",
				Senses: []kaikkiSense{
					{
						Glosses:  []string{"to move quickly"},
						Examples: []kaikkiExample{
							{Text: "He runs fast."},
							{Text: "She ran home."},
						},
					},
				},
			},
			want: 2.6, // 1.0 sense + 0.3 + 0.3 examples + 1.0 single-word
		},
		{
			name: "IPA pronunciation adds 2.0 once",
			entry: kaikkiEntry{
				Word: "dog",
				Sounds: []kaikkiSound{
					{IPA: "/dɒɡ/", Tags: []string{"UK"}},
				},
			},
			want: 3.0, // 2.0 IPA + 1.0 single-word
		},
		{
			name: "multiple IPA sounds still add only 2.0",
			entry: kaikkiEntry{
				Word: "dog",
				Sounds: []kaikkiSound{
					{IPA: "/dɒɡ/", Tags: []string{"UK"}},
					{IPA: "/dɑːɡ/", Tags: []string{"US"}},
				},
			},
			want: 3.0, // 2.0 IPA (once) + 1.0 single-word
		},
		{
			name: "sounds without IPA do not count",
			entry: kaikkiEntry{
				Word: "dog",
				Sounds: []kaikkiSound{
					{IPA: "", Tags: []string{"UK"}},
				},
			},
			want: 1.0, // only single-word bonus
		},
		{
			name: "multi-word entry gets no single-word bonus",
			entry: kaikkiEntry{
				Word: "give up",
				Senses: []kaikkiSense{
					{Glosses: []string{"to stop trying"}},
				},
			},
			want: 1.0, // 1.0 sense only, no single-word bonus
		},
		{
			name: "full rich entry",
			entry: kaikkiEntry{
				Word: "serendipity",
				Senses: []kaikkiSense{
					{
						Glosses: []string{"the occurrence of events by chance in a happy way"},
						Examples: []kaikkiExample{
							{Text: "A fortunate stroke of serendipity."},
							{Text: "By pure serendipity, they found the solution."},
						},
						Translations: []kaikkiTranslation{
							{Code: "ru", Word: "серендипность"},
							{Code: "de", Word: "Serendipitat"},
						},
					},
					{
						Glosses: []string{"the faculty of making happy discoveries"},
						Examples: []kaikkiExample{
							{Text: "She had a talent for serendipity."},
						},
						Translations: []kaikkiTranslation{
							{Code: "ru", Word: "интуиция"},
							{Code: "ru", Word: "счастливая находка"},
						},
					},
				},
				Sounds: []kaikkiSound{
					{IPA: "/ˌsɛɹ.ən.ˈdɪp.ə.ti/", Tags: []string{"US"}},
				},
			},
			// 2 senses: 2.0
			// 3 ru translations: 1.5
			// 3 examples: 0.9
			// IPA: 2.0
			// single-word: 1.0
			// total: 7.4
			want: 7.4,
		},
		{
			name: "multiple senses, some valid some empty",
			entry: kaikkiEntry{
				Word: "bank",
				Senses: []kaikkiSense{
					{Glosses: []string{"a financial institution"}},
					{Glosses: []string{}},
					{Glosses: []string{"the side of a river"}},
					{Glosses: nil},
				},
			},
			want: 3.0, // 2 valid senses (2.0) + 1.0 single-word
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreEntry(&tt.entry)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("ScoreEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}
