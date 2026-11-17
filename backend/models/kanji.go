package models

// KanjiEntry stores one level-classified character and its bilingual reference data.
type KanjiEntry struct {
	Base

	SeedKey           string `gorm:"type:varchar(100);not null;uniqueIndex" json:"-"`
	Character         string `gorm:"type:varchar(8);not null;uniqueIndex" json:"character"`
	JLPTLevel         string `gorm:"type:varchar(2);not null;index" json:"jlpt_level"`
	SortOrder         int    `gorm:"not null;default:0" json:"sort_order"`
	StrokeCount       int    `gorm:"not null;default:0" json:"stroke_count"`
	Grade             int    `gorm:"not null;default:0" json:"grade"`
	Frequency         int    `gorm:"not null;default:0" json:"frequency"`
	Radical           string `gorm:"type:varchar(8)" json:"radical"`
	OnReadings        string `gorm:"type:text" json:"-"`
	KunReadings       string `gorm:"type:text" json:"-"`
	EnglishMeaning    string `gorm:"type:text;not null" json:"english_meaning"`
	IndonesianMeaning string `gorm:"type:text;not null" json:"indonesian_meaning"`
	ExampleWords      string `gorm:"type:text" json:"-"`
	StrokeSVG         string `gorm:"type:varchar(100);not null" json:"stroke_svg"`
}
