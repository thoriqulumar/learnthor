package main

import "testing"

func TestKanaToRomaji(t *testing.T) {
	cases := []struct {
		kana string
		want string
	}{
		{"あ", "a"},
		{"こんにちは", "konnichiha"},
		{"ありがとう", "arigatou"},
		{"じゅう", "juu"},
		{"がっこう", "gakkou"},
		{"ちょっと", "chotto"},
		{"やった", "yatta"},
		{"きゃ", "kya"},
		{"しゅ", "shu"},
		{"ちょ", "cho"},
		{"りゅ", "ryu"},
		{"シャツ", "shatsu"},
		{"アクセサリー", "akusesarii"},
		{"パーティー", "paatii"},
		{"ティ", "ti"},
		{"ディ", "di"},
		{"ウィ", "wi"},
		{"ウェ", "we"},
		{"フォ", "fo"},
		{"", ""},
		{"english", "english"}, // latin passes through
		// 猫 (kanji) passes through, が → ga, 好 (kanji) passes through, き → ki
		{"猫が好き", "猫ga好ki"},
	}
	for _, c := range cases {
		got := KanaToRomaji(c.kana)
		if got != c.want {
			t.Errorf("KanaToRomaji(%q) = %q, want %q", c.kana, got, c.want)
		}
	}
}
