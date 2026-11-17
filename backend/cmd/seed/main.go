package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/learnthor/backend/db"
	"github.com/learnthor/backend/models"
)

func main() {
	catalogPath := flag.String("catalog", "", "catalog JSON path (defaults to embedded catalog)")
	kanjiCatalogPath := flag.String("kanji-catalog", "", "kanji catalog JSON path (defaults to embedded catalog)")
	jlptMaterialCatalogPath := flag.String("jlpt-material-catalog", "", "JLPT material catalog JSON path (defaults to embedded catalog)")
	validate := flag.Bool("validate", false, "validate catalog without connecting to a database")
	dryRun := flag.Bool("dry-run", false, "validate only; no database connection or writes")
	reset := flag.Bool("reset", false, "delete curriculum and learning progress before import")
	confirmReset := flag.Bool("confirm-reset", false, "confirm destruction of curriculum and progress")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatal("unexpected arguments")
	}
	if *reset && (!*confirmReset || *validate || *dryRun) {
		log.Fatal("--reset requires --confirm-reset and cannot be combined with validation/dry-run")
	}
	c, err := loadCatalog(*catalogPath)
	if err != nil {
		log.Fatalf("catalog validation: %v", err)
	}
	reportCatalog(c)
	kanjiCatalog, err := loadKanjiCatalog(*kanjiCatalogPath)
	if err != nil {
		log.Fatalf("kanji catalog validation: %v", err)
	}
	log.Printf("validated kanji catalog: %d entries", len(kanjiCatalog.Entries))
	jlptMaterialCatalog, err := loadJLPTMaterialCatalog(*jlptMaterialCatalogPath)
	if err != nil {
		log.Fatalf("JLPT material catalog validation: %v", err)
	}
	log.Printf("validated JLPT material catalog: %d lessons", len(jlptMaterialCatalog.Lessons))
	if *validate || *dryRun {
		log.Println("no database connection or changes made")
		return
	}
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from system environment")
	}
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := importCurriculum(database, c, *reset); err != nil {
		log.Fatalf("seed rolled back: %v", err)
	}
	if err := importKanjiCatalog(database, kanjiCatalog); err != nil {
		log.Fatalf("kanji seed rolled back: %v", err)
	}
	if err := importJLPTMaterialCatalog(database, jlptMaterialCatalog); err != nil {
		log.Fatalf("JLPT material seed rolled back: %v", err)
	}
	log.Println("seed complete (existing IDs and progress preserved unless reset explicitly requested)")
}

// resetCurriculum deletes all curriculum and progress rows while keeping
// the users table intact. Tables are truncated in dependency order to
// satisfy foreign-key constraints.
func resetCurriculum(db *gorm.DB) error {
	tables := []string{
		"srs_reviews",
		"session_items",
		"study_sessions",
		"module_skill_progress",
		"module_progress",
		"section_progress",
		"vocab_items",
		"modules",
		"sections",
		"courses",
	}
	for _, t := range tables {
		if err := db.Exec("TRUNCATE TABLE " + t + " RESTART IDENTITY CASCADE").Error; err != nil {
			return fmt.Errorf("truncate %s: %w", t, err)
		}
	}
	return nil
}

// strPtr converts a string literal to *string (for nullable Kanji field).
func strPtr(s string) *string { return &s }

// ─────────────────────────────────────────────────────────────────────────────
// Seed entry point
// ─────────────────────────────────────────────────────────────────────────────

func seed(db *gorm.DB, c catalog) {
	course := upsertCourse(db, models.Course{
		Title:       "JLPT N5 Masterclass",
		Description: "Complete preparation for the JLPT N5 exam. Covers hiragana, katakana, essential vocabulary, basic kanji, and foundational grammar.",
		SortOrder:   1,
	})

	seedHiragana(db, course)
	seedKatakana(db, course)
	seedGreetings(db, course)
	seedNumbers(db, course)
	seedBasicVocabulary(db, course)
	seedBasicKanji(db, course)
	seedCatalog(db, course, c)
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 1 — Hiragana
// ─────────────────────────────────────────────────────────────────────────────

func seedHiragana(database *gorm.DB, course models.Course) {
	sec := upsertSection(database, course.ID, models.Section{Title: "Hiragana", SortOrder: 1})

	m1 := upsertModule(database, sec.ID, models.Module{Title: "Vowels (あいうえお)", SortOrder: 1, PassThreshold: 90})
	upsertVocabItems(database, m1.ID, []vocabSeed{
		{nil, "あ", "a", "The letter 'a'", "character", ""},
		{nil, "い", "i", "The letter 'i'", "character", ""},
		{nil, "う", "u", "The letter 'u'", "character", ""},
		{nil, "え", "e", "The letter 'e'", "character", ""},
		{nil, "お", "o", "The letter 'o'", "character", ""},
	})

	m2 := upsertModule(database, sec.ID, models.Module{Title: "K-row (かきくけこ)", SortOrder: 2, PassThreshold: 90})
	upsertVocabItems(database, m2.ID, []vocabSeed{
		{nil, "か", "ka", "The letter 'ka'", "character", ""},
		{nil, "き", "ki", "The letter 'ki'", "character", ""},
		{nil, "く", "ku", "The letter 'ku'", "character", ""},
		{nil, "け", "ke", "The letter 'ke'", "character", ""},
		{nil, "こ", "ko", "The letter 'ko'", "character", ""},
	})

	m3 := upsertModule(database, sec.ID, models.Module{Title: "S-row (さしすせそ)", SortOrder: 3, PassThreshold: 90})
	upsertVocabItems(database, m3.ID, []vocabSeed{
		{nil, "さ", "sa", "The letter 'sa'", "character", ""},
		{nil, "し", "shi", "The letter 'shi'", "character", ""},
		{nil, "す", "su", "The letter 'su'", "character", ""},
		{nil, "せ", "se", "The letter 'se'", "character", ""},
		{nil, "そ", "so", "The letter 'so'", "character", ""},
	})

	m4 := upsertModule(database, sec.ID, models.Module{Title: "T-row (たちつてと)", SortOrder: 4, PassThreshold: 90})
	upsertVocabItems(database, m4.ID, []vocabSeed{
		{nil, "た", "ta", "The letter 'ta'", "character", ""},
		{nil, "ち", "chi", "The letter 'chi'", "character", ""},
		{nil, "つ", "tsu", "The letter 'tsu'", "character", ""},
		{nil, "て", "te", "The letter 'te'", "character", ""},
		{nil, "と", "to", "The letter 'to'", "character", ""},
	})

	m5 := upsertModule(database, sec.ID, models.Module{Title: "N & M rows (なにぬねの・まみむめも)", SortOrder: 5, PassThreshold: 90})
	upsertVocabItems(database, m5.ID, []vocabSeed{
		{nil, "な", "na", "The letter 'na'", "character", ""},
		{nil, "に", "ni", "The letter 'ni'", "character", ""},
		{nil, "ぬ", "nu", "The letter 'nu'", "character", ""},
		{nil, "ね", "ne", "The letter 'ne'", "character", ""},
		{nil, "の", "no", "The letter 'no'", "character", ""},
		{nil, "ま", "ma", "The letter 'ma'", "character", ""},
		{nil, "み", "mi", "The letter 'mi'", "character", ""},
		{nil, "む", "mu", "The letter 'mu'", "character", ""},
		{nil, "め", "me", "The letter 'me'", "character", ""},
		{nil, "も", "mo", "The letter 'mo'", "character", ""},
	})

	m6 := upsertModule(database, sec.ID, models.Module{Title: "H-row (はひふへほ)", SortOrder: 6, PassThreshold: 90})
	upsertVocabItems(database, m6.ID, []vocabSeed{
		{nil, "は", "ha", "The letter 'ha'", "character", ""},
		{nil, "ひ", "hi", "The letter 'hi'", "character", ""},
		{nil, "ふ", "fu", "The letter 'fu'", "character", ""},
		{nil, "へ", "he", "The letter 'he'", "character", ""},
		{nil, "ほ", "ho", "The letter 'ho'", "character", ""},
	})

	m7 := upsertModule(database, sec.ID, models.Module{Title: "Y & R rows (やゆよ・らりるれろ)", SortOrder: 7, PassThreshold: 90})
	upsertVocabItems(database, m7.ID, []vocabSeed{
		{nil, "や", "ya", "The letter 'ya'", "character", ""},
		{nil, "ゆ", "yu", "The letter 'yu'", "character", ""},
		{nil, "よ", "yo", "The letter 'yo'", "character", ""},
		{nil, "ら", "ra", "The letter 'ra'", "character", ""},
		{nil, "り", "ri", "The letter 'ri'", "character", ""},
		{nil, "る", "ru", "The letter 'ru'", "character", ""},
		{nil, "れ", "re", "The letter 're'", "character", ""},
		{nil, "ろ", "ro", "The letter 'ro'", "character", ""},
	})

	m8 := upsertModule(database, sec.ID, models.Module{Title: "W-row & N (わをん)", SortOrder: 8, PassThreshold: 90})
	upsertVocabItems(database, m8.ID, []vocabSeed{
		{nil, "わ", "wa", "The letter 'wa'", "character", ""},
		{nil, "を", "wo", "The particle 'wo/o'", "character", ""},
		{nil, "ん", "n", "The letter 'n'", "character", ""},
	})

	m9 := upsertModule(database, sec.ID, models.Module{Title: "Voiced Sounds — Dakuten (が・ざ・だ・ば rows)", SortOrder: 9, PassThreshold: 90})
	upsertVocabItems(database, m9.ID, []vocabSeed{
		{nil, "が", "ga", "The letter 'ga'", "character", ""},
		{nil, "ぎ", "gi", "The letter 'gi'", "character", ""},
		{nil, "ぐ", "gu", "The letter 'gu'", "character", ""},
		{nil, "げ", "ge", "The letter 'ge'", "character", ""},
		{nil, "ご", "go", "The letter 'go'", "character", ""},
		{nil, "ざ", "za", "The letter 'za'", "character", ""},
		{nil, "じ", "ji", "The letter 'ji'", "character", ""},
		{nil, "ず", "zu", "The letter 'zu'", "character", ""},
		{nil, "ぜ", "ze", "The letter 'ze'", "character", ""},
		{nil, "ぞ", "zo", "The letter 'zo'", "character", ""},
		{nil, "だ", "da", "The letter 'da'", "character", ""},
		{nil, "で", "de", "The letter 'de'", "character", ""},
		{nil, "ど", "do", "The letter 'do'", "character", ""},
		{nil, "ば", "ba", "The letter 'ba'", "character", ""},
		{nil, "び", "bi", "The letter 'bi'", "character", ""},
		{nil, "ぶ", "bu", "The letter 'bu'", "character", ""},
		{nil, "べ", "be", "The letter 'be'", "character", ""},
		{nil, "ぼ", "bo", "The letter 'bo'", "character", ""},
	})

	m10 := upsertModule(database, sec.ID, models.Module{Title: "Semi-voiced — Handakuten (ぱぴぷぺぽ)", SortOrder: 10, PassThreshold: 90})
	upsertVocabItems(database, m10.ID, []vocabSeed{
		{nil, "ぱ", "pa", "The letter 'pa'", "character", ""},
		{nil, "ぴ", "pi", "The letter 'pi'", "character", ""},
		{nil, "ぷ", "pu", "The letter 'pu'", "character", ""},
		{nil, "ぺ", "pe", "The letter 'pe'", "character", ""},
		{nil, "ぽ", "po", "The letter 'po'", "character", ""},
	})

	m11 := upsertModule(database, sec.ID, models.Module{Title: "Combination Sounds (きゃ・しゃ・ちゃ etc.)", SortOrder: 11, PassThreshold: 90})
	upsertVocabItems(database, m11.ID, []vocabSeed{
		{nil, "きゃ", "kya", "Combination 'kya'", "character", ""},
		{nil, "きゅ", "kyu", "Combination 'kyu'", "character", ""},
		{nil, "きょ", "kyo", "Combination 'kyo'", "character", ""},
		{nil, "しゃ", "sha", "Combination 'sha'", "character", ""},
		{nil, "しゅ", "shu", "Combination 'shu'", "character", ""},
		{nil, "しょ", "sho", "Combination 'sho'", "character", ""},
		{nil, "ちゃ", "cha", "Combination 'cha'", "character", ""},
		{nil, "ちゅ", "chu", "Combination 'chu'", "character", ""},
		{nil, "ちょ", "cho", "Combination 'cho'", "character", ""},
		{nil, "にゃ", "nya", "Combination 'nya'", "character", ""},
		{nil, "にゅ", "nyu", "Combination 'nyu'", "character", ""},
		{nil, "にょ", "nyo", "Combination 'nyo'", "character", ""},
		{nil, "ひゃ", "hya", "Combination 'hya'", "character", ""},
		{nil, "ひゅ", "hyu", "Combination 'hyu'", "character", ""},
		{nil, "ひょ", "hyo", "Combination 'hyo'", "character", ""},
		{nil, "みゃ", "mya", "Combination 'mya'", "character", ""},
		{nil, "みゅ", "myu", "Combination 'myu'", "character", ""},
		{nil, "みょ", "myo", "Combination 'myo'", "character", ""},
		{nil, "りゃ", "rya", "Combination 'rya'", "character", ""},
		{nil, "りゅ", "ryu", "Combination 'ryu'", "character", ""},
		{nil, "りょ", "ryo", "Combination 'ryo'", "character", ""},
		{nil, "ぎゃ", "gya", "Combination 'gya'", "character", ""},
		{nil, "ぎゅ", "gyu", "Combination 'gyu'", "character", ""},
		{nil, "ぎょ", "gyo", "Combination 'gyo'", "character", ""},
		{nil, "じゃ", "ja", "Combination 'ja'", "character", ""},
		{nil, "じゅ", "ju", "Combination 'ju'", "character", ""},
		{nil, "じょ", "jo", "Combination 'jo'", "character", ""},
		{nil, "びゃ", "bya", "Combination 'bya'", "character", ""},
		{nil, "びゅ", "byu", "Combination 'byu'", "character", ""},
		{nil, "びょ", "byo", "Combination 'byo'", "character", ""},
		{nil, "ぴゃ", "pya", "Combination 'pya'", "character", ""},
		{nil, "ぴゅ", "pyu", "Combination 'pyu'", "character", ""},
		{nil, "ぴょ", "pyo", "Combination 'pyo'", "character", ""},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 2 — Katakana
// ─────────────────────────────────────────────────────────────────────────────

func seedKatakana(database *gorm.DB, course models.Course) {
	sec := upsertSection(database, course.ID, models.Section{Title: "Katakana", SortOrder: 2})

	m1 := upsertModule(database, sec.ID, models.Module{Title: "Vowels (アイウエオ)", SortOrder: 1, PassThreshold: 90})
	upsertVocabItems(database, m1.ID, []vocabSeed{
		{nil, "ア", "a", "The letter 'a'", "character", ""},
		{nil, "イ", "i", "The letter 'i'", "character", ""},
		{nil, "ウ", "u", "The letter 'u'", "character", ""},
		{nil, "エ", "e", "The letter 'e'", "character", ""},
		{nil, "オ", "o", "The letter 'o'", "character", ""},
	})

	m2 := upsertModule(database, sec.ID, models.Module{Title: "K-row (カキクケコ)", SortOrder: 2, PassThreshold: 90})
	upsertVocabItems(database, m2.ID, []vocabSeed{
		{nil, "カ", "ka", "The letter 'ka'", "character", ""},
		{nil, "キ", "ki", "The letter 'ki'", "character", ""},
		{nil, "ク", "ku", "The letter 'ku'", "character", ""},
		{nil, "ケ", "ke", "The letter 'ke'", "character", ""},
		{nil, "コ", "ko", "The letter 'ko'", "character", ""},
	})

	m3 := upsertModule(database, sec.ID, models.Module{Title: "S-row (サシスセソ)", SortOrder: 3, PassThreshold: 90})
	upsertVocabItems(database, m3.ID, []vocabSeed{
		{nil, "サ", "sa", "The letter 'sa'", "character", ""},
		{nil, "シ", "shi", "The letter 'shi'", "character", ""},
		{nil, "ス", "su", "The letter 'su'", "character", ""},
		{nil, "セ", "se", "The letter 'se'", "character", ""},
		{nil, "ソ", "so", "The letter 'so'", "character", ""},
	})

	m4 := upsertModule(database, sec.ID, models.Module{Title: "T-row (タチツテト)", SortOrder: 4, PassThreshold: 90})
	upsertVocabItems(database, m4.ID, []vocabSeed{
		{nil, "タ", "ta", "The letter 'ta'", "character", ""},
		{nil, "チ", "chi", "The letter 'chi'", "character", ""},
		{nil, "ツ", "tsu", "The letter 'tsu'", "character", ""},
		{nil, "テ", "te", "The letter 'te'", "character", ""},
		{nil, "ト", "to", "The letter 'to'", "character", ""},
	})

	m5 := upsertModule(database, sec.ID, models.Module{Title: "N & M rows (ナニヌネノ・マミムメモ)", SortOrder: 5, PassThreshold: 90})
	upsertVocabItems(database, m5.ID, []vocabSeed{
		{nil, "ナ", "na", "The letter 'na'", "character", ""},
		{nil, "ニ", "ni", "The letter 'ni'", "character", ""},
		{nil, "ヌ", "nu", "The letter 'nu'", "character", ""},
		{nil, "ネ", "ne", "The letter 'ne'", "character", ""},
		{nil, "ノ", "no", "The letter 'no'", "character", ""},
		{nil, "マ", "ma", "The letter 'ma'", "character", ""},
		{nil, "ミ", "mi", "The letter 'mi'", "character", ""},
		{nil, "ム", "mu", "The letter 'mu'", "character", ""},
		{nil, "メ", "me", "The letter 'me'", "character", ""},
		{nil, "モ", "mo", "The letter 'mo'", "character", ""},
	})

	m6 := upsertModule(database, sec.ID, models.Module{Title: "H-row (ハヒフヘホ)", SortOrder: 6, PassThreshold: 90})
	upsertVocabItems(database, m6.ID, []vocabSeed{
		{nil, "ハ", "ha", "The letter 'ha'", "character", ""},
		{nil, "ヒ", "hi", "The letter 'hi'", "character", ""},
		{nil, "フ", "fu", "The letter 'fu'", "character", ""},
		{nil, "ヘ", "he", "The letter 'he'", "character", ""},
		{nil, "ホ", "ho", "The letter 'ho'", "character", ""},
	})

	m7 := upsertModule(database, sec.ID, models.Module{Title: "Y & R rows (ヤユヨ・ラリルレロ)", SortOrder: 7, PassThreshold: 90})
	upsertVocabItems(database, m7.ID, []vocabSeed{
		{nil, "ヤ", "ya", "The letter 'ya'", "character", ""},
		{nil, "ユ", "yu", "The letter 'yu'", "character", ""},
		{nil, "ヨ", "yo", "The letter 'yo'", "character", ""},
		{nil, "ラ", "ra", "The letter 'ra'", "character", ""},
		{nil, "リ", "ri", "The letter 'ri'", "character", ""},
		{nil, "ル", "ru", "The letter 'ru'", "character", ""},
		{nil, "レ", "re", "The letter 're'", "character", ""},
		{nil, "ロ", "ro", "The letter 'ro'", "character", ""},
	})

	m8 := upsertModule(database, sec.ID, models.Module{Title: "W-row & N (ワヲン)", SortOrder: 8, PassThreshold: 90})
	upsertVocabItems(database, m8.ID, []vocabSeed{
		{nil, "ワ", "wa", "The letter 'wa'", "character", ""},
		{nil, "ヲ", "wo", "The particle 'wo/o'", "character", ""},
		{nil, "ン", "n", "The letter 'n'", "character", ""},
	})

	m9 := upsertModule(database, sec.ID, models.Module{Title: "Voiced Sounds — Dakuten (ガ・ザ・ダ・バ rows)", SortOrder: 9, PassThreshold: 90})
	upsertVocabItems(database, m9.ID, []vocabSeed{
		{nil, "ガ", "ga", "The letter 'ga'", "character", ""},
		{nil, "ギ", "gi", "The letter 'gi'", "character", ""},
		{nil, "グ", "gu", "The letter 'gu'", "character", ""},
		{nil, "ゲ", "ge", "The letter 'ge'", "character", ""},
		{nil, "ゴ", "go", "The letter 'go'", "character", ""},
		{nil, "ザ", "za", "The letter 'za'", "character", ""},
		{nil, "ジ", "ji", "The letter 'ji'", "character", ""},
		{nil, "ズ", "zu", "The letter 'zu'", "character", ""},
		{nil, "ゼ", "ze", "The letter 'ze'", "character", ""},
		{nil, "ゾ", "zo", "The letter 'zo'", "character", ""},
		{nil, "ダ", "da", "The letter 'da'", "character", ""},
		{nil, "デ", "de", "The letter 'de'", "character", ""},
		{nil, "ド", "do", "The letter 'do'", "character", ""},
		{nil, "バ", "ba", "The letter 'ba'", "character", ""},
		{nil, "ビ", "bi", "The letter 'bi'", "character", ""},
		{nil, "ブ", "bu", "The letter 'bu'", "character", ""},
		{nil, "ベ", "be", "The letter 'be'", "character", ""},
		{nil, "ボ", "bo", "The letter 'bo'", "character", ""},
	})

	m10 := upsertModule(database, sec.ID, models.Module{Title: "Semi-voiced — Handakuten (パピプペポ)", SortOrder: 10, PassThreshold: 90})
	upsertVocabItems(database, m10.ID, []vocabSeed{
		{nil, "パ", "pa", "The letter 'pa'", "character", ""},
		{nil, "ピ", "pi", "The letter 'pi'", "character", ""},
		{nil, "プ", "pu", "The letter 'pu'", "character", ""},
		{nil, "ペ", "pe", "The letter 'pe'", "character", ""},
		{nil, "ポ", "po", "The letter 'po'", "character", ""},
	})

	m11 := upsertModule(database, sec.ID, models.Module{Title: "Combination Sounds (キャ・シャ・チャ etc.)", SortOrder: 11, PassThreshold: 90})
	upsertVocabItems(database, m11.ID, []vocabSeed{
		{nil, "キャ", "kya", "Combination 'kya'", "character", ""},
		{nil, "キュ", "kyu", "Combination 'kyu'", "character", ""},
		{nil, "キョ", "kyo", "Combination 'kyo'", "character", ""},
		{nil, "シャ", "sha", "Combination 'sha'", "character", ""},
		{nil, "シュ", "shu", "Combination 'shu'", "character", ""},
		{nil, "ショ", "sho", "Combination 'sho'", "character", ""},
		{nil, "チャ", "cha", "Combination 'cha'", "character", ""},
		{nil, "チュ", "chu", "Combination 'chu'", "character", ""},
		{nil, "チョ", "cho", "Combination 'cho'", "character", ""},
		{nil, "ニャ", "nya", "Combination 'nya'", "character", ""},
		{nil, "ニュ", "nyu", "Combination 'nyu'", "character", ""},
		{nil, "ニョ", "nyo", "Combination 'nyo'", "character", ""},
		{nil, "ヒャ", "hya", "Combination 'hya'", "character", ""},
		{nil, "ヒュ", "hyu", "Combination 'hyu'", "character", ""},
		{nil, "ヒョ", "hyo", "Combination 'hyo'", "character", ""},
		{nil, "ミャ", "mya", "Combination 'mya'", "character", ""},
		{nil, "ミュ", "myu", "Combination 'myu'", "character", ""},
		{nil, "ミョ", "myo", "Combination 'myo'", "character", ""},
		{nil, "リャ", "rya", "Combination 'rya'", "character", ""},
		{nil, "リュ", "ryu", "Combination 'ryu'", "character", ""},
		{nil, "リョ", "ryo", "Combination 'ryo'", "character", ""},
		{nil, "ギャ", "gya", "Combination 'gya'", "character", ""},
		{nil, "ギュ", "gyu", "Combination 'gyu'", "character", ""},
		{nil, "ギョ", "gyo", "Combination 'gyo'", "character", ""},
		{nil, "ジャ", "ja", "Combination 'ja'", "character", ""},
		{nil, "ジュ", "ju", "Combination 'ju'", "character", ""},
		{nil, "ジョ", "jo", "Combination 'jo'", "character", ""},
		{nil, "ビャ", "bya", "Combination 'bya'", "character", ""},
		{nil, "ビュ", "byu", "Combination 'byu'", "character", ""},
		{nil, "ビョ", "byo", "Combination 'byo'", "character", ""},
		{nil, "ピャ", "pya", "Combination 'pya'", "character", ""},
		{nil, "ピュ", "pyu", "Combination 'pyu'", "character", ""},
		{nil, "ピョ", "pyo", "Combination 'pyo'", "character", ""},
	})

	m12 := upsertModule(database, sec.ID, models.Module{Title: "Loanword Special Sounds (ファ・ティ・ウィ etc.)", SortOrder: 12, PassThreshold: 90})
	upsertVocabItems(database, m12.ID, []vocabSeed{
		{nil, "ファ", "fa", "Loanword sound 'fa' (e.g. ファン fan)", "character", ""},
		{nil, "フィ", "fi", "Loanword sound 'fi' (e.g. フィルム film)", "character", ""},
		{nil, "フェ", "fe", "Loanword sound 'fe' (e.g. フェスティバル festival)", "character", ""},
		{nil, "フォ", "fo", "Loanword sound 'fo' (e.g. フォーク fork)", "character", ""},
		{nil, "ティ", "ti", "Loanword sound 'ti' (e.g. パーティー party)", "character", ""},
		{nil, "ディ", "di", "Loanword sound 'di' (e.g. ディスコ disco)", "character", ""},
		{nil, "ウィ", "wi", "Loanword sound 'wi' (e.g. ウィスキー whisky)", "character", ""},
		{nil, "ウェ", "we", "Loanword sound 'we' (e.g. ウェブ web)", "character", ""},
		{nil, "ウォ", "wo2", "Loanword sound 'wo' (e.g. ウォーター water)", "character", ""},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 3 — Basic Greetings & Expressions
// ─────────────────────────────────────────────────────────────────────────────

func seedGreetings(database *gorm.DB, course models.Course) {
	sec := upsertSection(database, course.ID, models.Section{Title: "Basic Greetings & Expressions", SortOrder: 3})

	m1 := upsertModule(database, sec.ID, models.Module{Title: "Daily Greetings", SortOrder: 1, PassThreshold: 90})
	upsertVocabItems(database, m1.ID, []vocabSeed{
		{nil, "おはよう", "ohayou", "Good morning (casual)", "expression", "おはよう！"},
		{nil, "おはようございます", "ohayou gozaimasu", "Good morning (formal)", "expression", "おはようございます、せんせい。"},
		{nil, "こんにちは", "konnichiwa", "Hello / Good afternoon", "expression", "こんにちは、おげんきですか？"},
		{nil, "こんばんは", "konbanwa", "Good evening", "expression", "こんばんは！"},
		{nil, "おやすみなさい", "oyasuminasai", "Good night", "expression", "おやすみなさい。"},
		{nil, "さようなら", "sayounara", "Goodbye", "expression", "さようなら、またね。"},
		{nil, "ありがとうございます", "arigatou gozaimasu", "Thank you (formal)", "expression", "ありがとうございます！"},
		{nil, "すみません", "sumimasen", "Excuse me / I'm sorry", "expression", "すみません、トイレはどこですか？"},
	})

	m2 := upsertModule(database, sec.ID, models.Module{Title: "Introductions", SortOrder: 2, PassThreshold: 90})
	upsertVocabItems(database, m2.ID, []vocabSeed{
		{nil, "はじめまして", "hajimemashite", "Nice to meet you", "expression", "はじめまして、たなかです。"},
		{nil, "どうぞよろしく", "douzo yoroshiku", "Please be kind to me", "expression", "どうぞよろしくおねがいします。"},
		{nil, "わたし", "watashi", "I / Me", "pronoun", "わたしはがくせいです。"},
		{nil, "なまえ", "namae", "Name", "noun", "なまえはなんですか？"},
		{nil, "にほんご", "nihongo", "Japanese language", "noun", "にほんごをべんきょうします。"},
		{nil, "えいご", "eigo", "English language", "noun", "えいごがはなせます。"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 3 — Numbers & Time
// ─────────────────────────────────────────────────────────────────────────────

func seedNumbers(database *gorm.DB, course models.Course) {
	sec := upsertSection(database, course.ID, models.Section{Title: "Numbers & Time", SortOrder: 4})

	m1 := upsertModule(database, sec.ID, models.Module{Title: "Numbers 1–10", SortOrder: 1, PassThreshold: 90})
	upsertVocabItems(database, m1.ID, []vocabSeed{
		{strPtr("一"), "いち", "ichi", "One", "number", "いちにさん…"},
		{strPtr("二"), "に", "ni", "Two", "number", ""},
		{strPtr("三"), "さん", "san", "Three", "number", ""},
		{strPtr("四"), "し・よん", "shi / yon", "Four", "number", ""},
		{strPtr("五"), "ご", "go", "Five", "number", ""},
		{strPtr("六"), "ろく", "roku", "Six", "number", ""},
		{strPtr("七"), "しち・なな", "shichi / nana", "Seven", "number", ""},
		{strPtr("八"), "はち", "hachi", "Eight", "number", ""},
		{strPtr("九"), "く・きゅう", "ku / kyuu", "Nine", "number", ""},
		{strPtr("十"), "じゅう", "juu", "Ten", "number", "じゅうえんです。"},
	})

	m2 := upsertModule(database, sec.ID, models.Module{Title: "Days of the Week", SortOrder: 2, PassThreshold: 90})
	upsertVocabItems(database, m2.ID, []vocabSeed{
		{strPtr("月曜日"), "げつようび", "getsuyoubi", "Monday", "noun", "げつようびにがっこうへいきます。"},
		{strPtr("火曜日"), "かようび", "kayoubi", "Tuesday", "noun", ""},
		{strPtr("水曜日"), "すいようび", "suiyoubi", "Wednesday", "noun", ""},
		{strPtr("木曜日"), "もくようび", "mokuyoubi", "Thursday", "noun", ""},
		{strPtr("金曜日"), "きんようび", "kinyoubi", "Friday", "noun", "きんようびがすきです。"},
		{strPtr("土曜日"), "どようび", "doyoubi", "Saturday", "noun", ""},
		{strPtr("日曜日"), "にちようび", "nichiyoubi", "Sunday", "noun", "にちようびはやすみです。"},
	})

	m3 := upsertModule(database, sec.ID, models.Module{Title: "Time Expressions", SortOrder: 3, PassThreshold: 90})
	upsertVocabItems(database, m3.ID, []vocabSeed{
		{strPtr("今"), "いま", "ima", "Now", "noun", "いまなんじですか？"},
		{strPtr("今日"), "きょう", "kyou", "Today", "noun", "きょうはいいてんきです。"},
		{strPtr("明日"), "あした", "ashita", "Tomorrow", "noun", "あしたがっこうがあります。"},
		{strPtr("昨日"), "きのう", "kinou", "Yesterday", "noun", "きのうえいがをみました。"},
		{strPtr("朝"), "あさ", "asa", "Morning", "noun", "あさごはんをたべます。"},
		{strPtr("昼"), "ひる", "hiru", "Noon / Daytime", "noun", "ひるごはんはなんですか？"},
		{strPtr("夜"), "よる", "yoru", "Night / Evening", "noun", "よるべんきょうします。"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 4 — Basic Vocabulary
// ─────────────────────────────────────────────────────────────────────────────

func seedBasicVocabulary(database *gorm.DB, course models.Course) {
	sec := upsertSection(database, course.ID, models.Section{Title: "Basic Vocabulary", SortOrder: 5})

	m1 := upsertModule(database, sec.ID, models.Module{Title: "Family Members", SortOrder: 1, PassThreshold: 90})
	upsertVocabItems(database, m1.ID, []vocabSeed{
		{strPtr("家族"), "かぞく", "kazoku", "Family", "noun", "わたしのかぞくはよにんです。"},
		{strPtr("父"), "ちち", "chichi", "My father", "noun", "ちちはいしゃです。"},
		{strPtr("母"), "はは", "haha", "My mother", "noun", "はははせんせいです。"},
		{strPtr("お父さん"), "おとうさん", "otousan", "Father (someone else's)", "noun", "おとうさんはどこですか？"},
		{strPtr("お母さん"), "おかあさん", "okaasan", "Mother (someone else's)", "noun", ""},
		{strPtr("兄"), "あに", "ani", "My older brother", "noun", ""},
		{strPtr("姉"), "あね", "ane", "My older sister", "noun", ""},
		{strPtr("弟"), "おとうと", "otouto", "Younger brother", "noun", ""},
		{strPtr("妹"), "いもうと", "imouto", "Younger sister", "noun", ""},
	})

	m2 := upsertModule(database, sec.ID, models.Module{Title: "Food & Drink", SortOrder: 2, PassThreshold: 90})
	upsertVocabItems(database, m2.ID, []vocabSeed{
		{strPtr("御飯"), "ごはん", "gohan", "Rice / Meal", "noun", "ごはんをたべます。"},
		{strPtr("水"), "みず", "mizu", "Water", "noun", "みずをのみます。"},
		{nil, "おちゃ", "ocha", "Green tea", "noun", "おちゃがすきです。"},
		{nil, "パン", "pan", "Bread", "noun", "あさパンをたべます。"},
		{strPtr("卵"), "たまご", "tamago", "Egg", "noun", "たまごりょうりがすきです。"},
		{strPtr("魚"), "さかな", "sakana", "Fish", "noun", "さかなをたべます。"},
		{strPtr("肉"), "にく", "niku", "Meat", "noun", "にくがすきです。"},
		{strPtr("野菜"), "やさい", "yasai", "Vegetables", "noun", "やさいはからだにいいです。"},
		{strPtr("果物"), "くだもの", "kudamono", "Fruit", "noun", "くだものをまいにちたべます。"},
		{strPtr("お酒"), "おさけ", "osake", "Alcohol / Japanese sake", "noun", "おさけはのみません。"},
	})

	m3 := upsertModule(database, sec.ID, models.Module{Title: "Colors", SortOrder: 3, PassThreshold: 90})
	upsertVocabItems(database, m3.ID, []vocabSeed{
		{strPtr("赤"), "あか", "aka", "Red", "noun/adjective", "あかいくるまです。"},
		{strPtr("青"), "あお", "ao", "Blue / Green", "noun/adjective", "そらはあおいです。"},
		{strPtr("白"), "しろ", "shiro", "White", "noun/adjective", "しろいねこがいます。"},
		{strPtr("黒"), "くろ", "kuro", "Black", "noun/adjective", "くろいかばんをもっています。"},
		{strPtr("黄色"), "きいろ", "kiiro", "Yellow", "noun/adjective", "きいろいはなです。"},
		{nil, "みどり", "midori", "Green", "noun/adjective", "みどりがすきです。"},
	})

	m4 := upsertModule(database, sec.ID, models.Module{Title: "Verbs: Daily Actions", SortOrder: 4, PassThreshold: 90})
	upsertVocabItems(database, m4.ID, []vocabSeed{
		{strPtr("食べる"), "たべる", "taberu", "To eat", "verb", "ごはんをたべます。"},
		{strPtr("飲む"), "のむ", "nomu", "To drink", "verb", "みずをのみます。"},
		{strPtr("見る"), "みる", "miru", "To see / watch", "verb", "テレビをみます。"},
		{strPtr("聞く"), "きく", "kiku", "To listen / ask", "verb", "おんがくをききます。"},
		{strPtr("話す"), "はなす", "hanasu", "To speak", "verb", "にほんごをはなします。"},
		{strPtr("書く"), "かく", "kaku", "To write", "verb", "てがみをかきます。"},
		{strPtr("読む"), "よむ", "yomu", "To read", "verb", "ほんをよみます。"},
		{strPtr("行く"), "いく", "iku", "To go", "verb", "がっこうへいきます。"},
		{strPtr("来る"), "くる", "kuru", "To come", "verb", "ともだちがきます。"},
		{strPtr("帰る"), "かえる", "kaeru", "To return home", "verb", "うちにかえります。"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 5 — Basic Kanji
// ─────────────────────────────────────────────────────────────────────────────

func seedBasicKanji(database *gorm.DB, course models.Course) {
	sec := upsertSection(database, course.ID, models.Section{Title: "Basic Kanji", SortOrder: 6})

	m1 := upsertModule(database, sec.ID, models.Module{Title: "Nature Kanji", SortOrder: 1, PassThreshold: 90})
	upsertVocabItems(database, m1.ID, []vocabSeed{
		{strPtr("山"), "やま", "yama", "Mountain", "noun", "やまがたかいです。"},
		{strPtr("川"), "かわ", "kawa", "River", "noun", "かわでおよぎます。"},
		{strPtr("木"), "き", "ki", "Tree", "noun", "きがおおきいです。"},
		{strPtr("火"), "ひ", "hi", "Fire", "noun", "ひがあついです。"},
		{strPtr("水"), "みず", "mizu", "Water", "noun", "みずをのみます。"},
		{strPtr("土"), "つち", "tsuchi", "Soil / Earth", "noun", "つちがやわらかいです。"},
		{strPtr("空"), "そら", "sora", "Sky", "noun", "そらがあおいです。"},
		{strPtr("花"), "はな", "hana", "Flower", "noun", "はながきれいです。"},
	})

	m2 := upsertModule(database, sec.ID, models.Module{Title: "People & Body Kanji", SortOrder: 2, PassThreshold: 90})
	upsertVocabItems(database, m2.ID, []vocabSeed{
		{strPtr("人"), "ひと", "hito", "Person", "noun", "あのひとはだれですか？"},
		{strPtr("女"), "おんな", "onna", "Woman / Female", "noun", "おんなのひとです。"},
		{strPtr("男"), "おとこ", "otoko", "Man / Male", "noun", "おとこのこです。"},
		{strPtr("子"), "こ", "ko", "Child", "noun", "こどもがすきです。"},
		{strPtr("目"), "め", "me", "Eye", "noun", "めがおおきいです。"},
		{strPtr("耳"), "みみ", "mimi", "Ear", "noun", "みみがいたいです。"},
		{strPtr("口"), "くち", "kuchi", "Mouth", "noun", "くちをあけてください。"},
		{strPtr("手"), "て", "te", "Hand", "noun", "てをあらいます。"},
		{strPtr("足"), "あし", "ashi", "Foot / Leg", "noun", "あしがながいです。"},
	})

	m3 := upsertModule(database, sec.ID, models.Module{Title: "Place & Direction Kanji", SortOrder: 3, PassThreshold: 90})
	upsertVocabItems(database, m3.ID, []vocabSeed{
		{strPtr("上"), "うえ", "ue", "Above / Up", "noun/adj", "つくえのうえにあります。"},
		{strPtr("下"), "した", "shita", "Below / Down", "noun/adj", "いすのしたにあります。"},
		{strPtr("右"), "みぎ", "migi", "Right", "noun", "みぎにまがってください。"},
		{strPtr("左"), "ひだり", "hidari", "Left", "noun", "ひだりにまがってください。"},
		{strPtr("中"), "なか", "naka", "Inside / Middle", "noun", "はこのなかにあります。"},
		{strPtr("外"), "そと", "soto", "Outside", "noun", "そとはさむいです。"},
		{strPtr("前"), "まえ", "mae", "In front / Before", "noun", "えきのまえです。"},
		{strPtr("後"), "うしろ", "ushiro", "Behind / After", "noun", "うしろをみてください。"},
	})
}
