package models

import "github.com/google/uuid"

// QuizType identifies the kind of question presented to the learner.
type QuizType string

const (
	// QuizTypeKanjiToKana shows the kanji/katakana written form; the learner
	// picks the correct hiragana (or katakana for loanwords) reading.
	QuizTypeKanjiToKana QuizType = "kanji_to_kana"

	// QuizTypeKanaToKanji shows the hiragana/katakana reading; the learner
	// picks the correct kanji from visually-similar options.
	QuizTypeKanaToKanji QuizType = "kana_to_kanji"

	// QuizTypeFillBlank shows an example sentence with the target word replaced
	// by ＿＿＿; the learner selects the correct kanji from similar options.
	QuizTypeFillBlank QuizType = "fill_blank"
)

// QuizOption is a single selectable answer in a multiple-choice question.
type QuizOption struct {
	// ID is the VocabItem.ID this option represents, used to submit answers.
	ID uuid.UUID `json:"id"`
	// Text is the string displayed to the learner.
	Text string `json:"text"`
}

// QuizQuestion is one multiple-choice question generated from a VocabItem.
type QuizQuestion struct {
	// VocabItemID identifies the vocabulary item under test.
	VocabItemID uuid.UUID `json:"vocab_item_id"`
	// Type describes how the question is framed.
	Type QuizType `json:"type"`
	// Prompt is what the learner reads:
	//   kanji_to_kana → kanji (or katakana) written form
	//   kana_to_kanji → hiragana/katakana reading
	//   fill_blank    → example sentence with target word replaced by ＿＿＿
	Prompt string `json:"prompt"`
	// CorrectID is the ID of the correct QuizOption.
	CorrectID uuid.UUID `json:"correct_id"`
	// Options is a shuffled list of up to four possible answers.
	Options []QuizOption `json:"options"`
}
