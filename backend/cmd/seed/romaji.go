package main

import "strings"

// kanaRomaji maps a single kana (hiragana or katakana) to its romaji.
// Covers gojūon, dakuten, handakuten. Yōon (small y-kana) and loanword
// contractions (small vowel kana) are handled by KanaToRomaji via the
// compound map below.
var kanaRomaji = map[rune]string{
	// ── Hiragana vowels ──
	'あ': "a", 'い': "i", 'う': "u", 'え': "e", 'お': "o",
	// ── Katakana vowels ──
	'ア': "a", 'イ': "i", 'ウ': "u", 'エ': "e", 'オ': "o",

	// ── K-row ──
	'か': "ka", 'き': "ki", 'く': "ku", 'け': "ke", 'こ': "ko",
	'カ': "ka", 'キ': "ki", 'ク': "ku", 'ケ': "ke", 'コ': "ko",

	// ── S-row (し = shi) ──
	'さ': "sa", 'し': "shi", 'す': "su", 'せ': "se", 'そ': "so",
	'サ': "sa", 'シ': "shi", 'ス': "su", 'セ': "se", 'ソ': "so",

	// ── T-row (ち = chi, つ = tsu) ──
	'た': "ta", 'ち': "chi", 'つ': "tsu", 'て': "te", 'と': "to",
	'タ': "ta", 'チ': "chi", 'ツ': "tsu", 'テ': "te", 'ト': "to",

	// ── N-row ──
	'な': "na", 'に': "ni", 'ぬ': "nu", 'ね': "ne", 'の': "no",
	'ナ': "na", 'ニ': "ni", 'ヌ': "nu", 'ネ': "ne", 'ノ': "no",

	// ── H-row (ふ = fu) ──
	'は': "ha", 'ひ': "hi", 'ふ': "fu", 'へ': "he", 'ほ': "ho",
	'ハ': "ha", 'ヒ': "hi", 'フ': "fu", 'ヘ': "he", 'ホ': "ho",

	// ── M-row ──
	'ま': "ma", 'み': "mi", 'む': "mu", 'め': "me", 'も': "mo",
	'マ': "ma", 'ミ': "mi", 'ム': "mu", 'メ': "me", 'モ': "mo",

	// ── Y-row ──
	'や': "ya", 'ゆ': "yu", 'よ': "yo",
	'ヤ': "ya", 'ユ': "yu", 'ヨ': "yo",

	// ── R-row ──
	'ら': "ra", 'り': "ri", 'る': "ru", 'れ': "re", 'ろ': "ro",
	'ラ': "ra", 'リ': "ri", 'ル': "ru", 'レ': "re", 'ロ': "ro",

	// ── W-row & N ──
	'わ': "wa", 'を': "wo", 'ん': "n",
	'ワ': "wa", 'ヲ': "wo", 'ン': "n",

	// ── Dakuten ──
	'が': "ga", 'ぎ': "gi", 'ぐ': "gu", 'げ': "ge", 'ご': "go",
	'ガ': "ga", 'ギ': "gi", 'グ': "gu", 'ゲ': "ge", 'ゴ': "go",
	'ざ': "za", 'じ': "ji", 'ず': "zu", 'ぜ': "ze", 'ぞ': "zo",
	'ザ': "za", 'ジ': "ji", 'ズ': "zu", 'ゼ': "ze", 'ゾ': "zo",
	'だ': "da", 'ぢ': "ji", 'づ': "zu", 'で': "de", 'ど': "do",
	'ダ': "da", 'ヂ': "ji", 'ヅ': "zu", 'デ': "de", 'ド': "do",
	'ば': "ba", 'び': "bi", 'ぶ': "bu", 'べ': "be", 'ぼ': "bo",
	'バ': "ba", 'ビ': "bi", 'ブ': "bu", 'ベ': "be", 'ボ': "bo",

	// ── Handakuten ──
	'ぱ': "pa", 'ぴ': "pi", 'ぷ': "pu", 'ぺ': "pe", 'ぽ': "po",
	'パ': "pa", 'ピ': "pi", 'プ': "pu", 'ペ': "pe", 'ポ': "po",
}

// yōonRomaji maps a two-kana yōon (base + small y) directly to its romaji,
// e.g. "しゃ" → "sha". Hiragana is keyed here; katakana equivalents
// (シャ → sha) are handled via the shared y consonant.
var yōonRomaji = map[string]string{
	// ── K-row ──
	"きゃ": "kya", "きゅ": "kyu", "きょ": "kyo",
	"キャ": "kya", "キュ": "kyu", "キョ": "kyo",
	// ── S-row (shi → sh-) ──
	"しゃ": "sha", "しゅ": "shu", "しょ": "sho",
	"シャ": "sha", "シュ": "shu", "ショ": "sho",
	// ── T-row (chi → ch-) ──
	"ちゃ": "cha", "ちゅ": "chu", "ちょ": "cho",
	"チャ": "cha", "チュ": "chu", "チョ": "cho",
	// ── N-row ──
	"にゃ": "nya", "にゅ": "nyu", "にょ": "nyo",
	"ニャ": "nya", "ニュ": "nyu", "ニョ": "nyo",
	// ── H-row ──
	"ひゃ": "hya", "ひゅ": "hyu", "ひょ": "hyo",
	"ヒャ": "hya", "ヒュ": "hyu", "ヒョ": "hyo",
	// ── M-row ──
	"みゃ": "mya", "みゅ": "myu", "みょ": "myo",
	"ミャ": "mya", "ミュ": "myu", "ミョ": "myo",
	// ── R-row ──
	"りゃ": "rya", "りゅ": "ryu", "りょ": "ryo",
	"リャ": "rya", "リュ": "ryu", "リョ": "ryo",
	// ── G-row ──
	"ぎゃ": "gya", "ぎゅ": "gyu", "ぎょ": "gyo",
	"ギャ": "gya", "ギュ": "gyu", "ギョ": "gyo",
	// ── Z-row (ji → j-) ──
	"じゃ": "ja", "じゅ": "ju", "じょ": "jo",
	"ジャ": "ja", "ジュ": "ju", "ジョ": "jo",
	// ── B-row ──
	"びゃ": "bya", "びゅ": "byu", "びょ": "byo",
	"ビャ": "bya", "ビュ": "byu", "ビョ": "byo",
	// ── P-row ──
	"ぴゃ": "pya", "ぴゅ": "pyu", "ぴょ": "pyo",
	"ピャ": "pya", "ピュ": "pyu", "ピョ": "pyo",
}

// loanwordCombo maps a two-kana katakana contraction (base + small vowel)
// used in loanwords to the romaji sound it represents, e.g. ティ → "ti".
var loanwordCombo = map[string]string{
	// ── W-row (modern loanwords) ──
	"ウィ": "wi", "ウェ": "we", "ウォ": "wo", "ウァ": "wa",
	// ── F-row (フ + small vowel) ──
	"ファ": "fa", "フィ": "fi", "フェ": "fe", "フォ": "fo",
	// ── T/D small-vowel contractions ──
	"ティ": "ti", "ディ": "di", "トゥ": "tu", "ドゥ": "du",
	// ── Smaller loanword combos occasionally seen ──
	"クァ": "kwa", "クィ": "kwi", "クェ": "kwe", "クォ": "kwo",
	"スィ": "si", "ズィ": "zi",
	"チェ": "che", "ツィ": "tsi", "ジェ": "je", "シェ": "she",
}

// KanaToRomaji transliterates a hiragana/katakana string to romaji.
// It is intentionally minimal — enough to enrich seed vocab batches
// that only carry a "reading" field and lack the richer romaji column.
// Any unmapped character (kanji, latin, punctuation) passes through
// unchanged so the function stays safe to call on mixed input.
func KanaToRomaji(kana string) string {
	if kana == "" {
		return ""
	}
	var b strings.Builder
	runes := []rune(kana)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Long vowel mark (ー): lengthen the previous vowel.
		if r == 'ー' {
			if b.Len() > 0 {
				if prev := b.String()[b.Len()-1]; isVowel(prev) {
					b.WriteByte(prev)
				}
			}
			continue
		}

		// Small tsu gemination: double the leading consonant of the next kana.
		if r == 'っ' || r == 'ッ' {
			if i+1 < len(runes) {
				if s, ok := kanaRomaji[runes[i+1]]; ok && len(s) > 0 {
					b.WriteByte(s[0])
				} else if i+2 < len(runes) {
					// Next char may be the start of a yōon, e.g. っきゃ → "kkya".
					pair := string(runes[i+1 : i+3])
					if s, ok := yōonRomaji[pair]; ok && len(s) > 0 {
						b.WriteByte(s[0])
					} else if s, ok := loanwordCombo[pair]; ok && len(s) > 0 {
						b.WriteByte(s[0])
					}
				}
			}
			continue
		}

		// Two-kana yōon (e.g. きゃ, シュ): look ahead one rune.
		if i+1 < len(runes) {
			pair := string(runes[i : i+2])
			if s, ok := yōonRomaji[pair]; ok {
				b.WriteString(s)
				i++
				continue
			}
			if s, ok := loanwordCombo[pair]; ok {
				b.WriteString(s)
				i++
				continue
			}
		}

		if s, ok := kanaRomaji[r]; ok {
			b.WriteString(s)
			continue
		}

		// Unmapped rune (kanji, latin, punctuation): pass through intact.
		b.WriteRune(r)
	}
	return b.String()
}

func isVowel(c byte) bool {
	return c == 'a' || c == 'i' || c == 'u' || c == 'e' || c == 'o'
}
