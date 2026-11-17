# Curriculum imports

## JLPT learning materials

`data/catalog/jlpt-materials.json` is the canonical catalog for the material
shown inside the N5, N4, and N3 level hubs. It currently contains 56 bilingual
lessons, including 47 particle lessons with 141 questions. Grammar, Reading /
Dokkai, and Exam Practice retain their existing lessons for each level.
Each lesson contains structured English and Indonesian explanations, Japanese
examples or a reading passage, and multiple-choice questions. Quiz
answers are graded by the backend and the learner's best score is stored in
`jlpt_lesson_progresses`.

The catalog is validated and imported with the other seed data by default. An
external catalog can be selected with:

```sh
go run ./cmd/seed --jlpt-material-catalog ./cmd/seed/data/catalog/jlpt-materials.json --validate
```

N5 material is available to authenticated learners. N4 and N3 material follows
the existing premium rules. Import identity comes from the immutable lesson and
question IDs, so wording and translations can be revised without losing quiz
progress.

The particle expansion uses an editorial progression, because JLPT does not
publish exhaustive grammar lists by level. N5 covers basic case, listing,
comparison, restriction, and sentence-ending particles; N4 extends connectors
and related uses; N3 adds focus, compound particles, and contrast. Higher-level
study also requires the earlier material. The authored source is
`expand-particles.cjs`; its stable IDs must be preserved when editing.

Backups before the October 8 particle expansion:
`backups/particles-before-expansion-20261008.zip` (seed JSON) and
`backups/particles-database-before-expansion-20261008.zip` (PostgreSQL dump).

### Backup taken before the level and material redesign

- `backups/before-jlpt-levels-20260930.zip` contains the application and seed
  source before the change. SHA-256:
  `4430A59F35C0AF4BDE447D1970E5C0614747D4A315BCB8983210D3F7696D4EAB`.
- `backups/database-before-jlpt-levels-20260930.zip` contains the complete
  PostgreSQL SQL dump taken before migration and import. SHA-256:
  `A2000DCD37BDC375EE88FD6C1C15682D82749221C0148247D9FC0F64D6EDC449`.

## Kanji catalog

`data/catalog/kanji.json` is the canonical N5-N3 kanji catalog. It contains
640 characters: 84 N5, 169 N4, and 387 N3. Each entry has a stable seed key,
JLPT level, stroke count, on/kun readings, English and Indonesian meanings,
example words, and a local KanjiVG stroke-order SVG path.

The catalog is imported with the curriculum by default. Validate or select it
explicitly with:

```sh
go run ./cmd/seed --validate
go run ./cmd/seed --kanji-catalog ./cmd/seed/data/catalog/kanji.json --validate
go run ./cmd/seed
```

Kanji imports update or insert by stable seed key and do not delete existing
rows. The source snapshots in `data/kanji-source/` come from OpenJLPT, which
enriches its community JLPT lists with KANJIDIC2 data and is distributed under
CC BY-SA 4.0. Stroke-order diagrams in `frontend/public/kanji/` come from
KanjiVG under CC BY-SA 3.0; its `COPYING` file is kept with the assets.

`build-kanji-catalog.cjs` rebuilds the catalog from those snapshots.
`translate-kanji.py` produced the first Indonesian glosses with an offline
Argos Translate model. These translations are useful as initial learning data,
but should be reviewed by a fluent editor as the catalog evolves.

### Backup taken before the kanji segment

- `backups/before-kanji-segment-20260930.zip` contains the application source
  and seed data before this feature. SHA-256:
  `A68F2FF2C05E6D95A2BEDE4618942E4838C61CDE0BD4C705F011BCBE674E7C26`.
- `backups/database-before-kanji-segment-20260930.zip` contains the PostgreSQL
  SQL dump made immediately before the schema migration and kanji import.
  SHA-256:
  `A16FC668EAA5FC08BA56F517E17CEC7EDBEF6911002AA5C569671108DBE95A8A`.

Keep both archives together: the first restores the code and seed source, and
the second restores the matching application database.

Edit **data/catalog/curriculum.json** for JSON-backed curriculum updates. It is
the canonical schema-version-1 catalog. The older JSON files remain unchanged
as source references; the importer no longer reads them. The six beginner
sections are still defined in Go and imported through the same persistence code.

## Backup taken before this change

The topic-based curriculum cleanup is backed up in two archives at the
repository root:

- `backups/seed-before-curriculum-cleanup-20260929.zip` contains the seed source
  and catalog before the cleanup. SHA-256:
  `94968B42B9C456900F5555417D0AAB5C9FD54853141C62981E83B8AE47A00C24`.
- `backups/database-before-curriculum-cleanup-20260929.zip` contains a complete
  PostgreSQL SQL dump made immediately before the import. SHA-256:
  `80E46112AA441834F8DBB71593A32F9DA3C83D7AD68AA49F35FB0AE81E0EB6FE`.

These backups allow the source catalog and the application data to be restored
independently.

### Earlier importer backup

`backups/seed-before-improvements-20260928-202418.zip` at the repository root
contains the original 214 JSON files, seed source, models, migration code,
Dockerfile, Go dependency manifests, Compose configuration, and deployment/status
documentation. SHA-256:

`D9F72F020B98169BA13648FC5FF3CD5646735C66DD69067BC9350402A8945A1F`

This is a source backup, not a PostgreSQL dump. No application database was
modified during development. Before applying imports to an existing deployment,
take and verify a database backup using your deployment's normal procedure.

## Validate and import

From `backend/`:

```sh
go run ./cmd/seed --validate
go run ./cmd/seed --dry-run
go run ./cmd/seed --catalog ./cmd/seed/data/catalog/curriculum.json --validate
go run ./cmd/seed
```

`--validate` and `--dry-run` are equivalent offline validation commands. They
report catalog totals without connecting to the database; they do not calculate
a diff against existing database rows. Normal imports require `DATABASE_URL`.

The default catalog is embedded in the compiled seeder. Rebuild the binary or
container after editing it, or provide an external file using `--catalog`.
Working directory and build-time source paths do not affect bundled imports.

```sh
docker compose build seed
docker compose run --rm seed --validate
docker compose run --rm seed
```

Normal imports insert new records and update existing records in place. They
never delete records absent from the catalog. Catalog entries in
`retired_sections` and `retired_modules` are hidden from learners while their
database IDs and all progress/history references are retained. Bringing an ID
back into the active catalog publishes it again. Entire curriculum imports run
in one transaction, including an explicitly requested reset; failures roll
back. PostgreSQL advisory locking serializes concurrent imports. Startup schema
migration runs separately, before that transaction.

Only for an intentional destructive reset of curriculum **and learner progress**:

```sh
go run ./cmd/seed --reset --confirm-reset
```

Users and payments survive resets; study sessions, reviews, and scores do not.
Do not use resets for routine content updates.

## Editing rules

- Keep every section, module, and item `id` unchanged after first import, even
  when changing spelling, translations, titles, or ordering. IDs must be unique
  within their entity kind across the entire catalog. Assign a new unique ID to
  genuinely new content. Never reuse retired IDs.
- IDs are stored in nullable unique `seed_key` columns. The original database
  UUID is preserved, retaining progress/review references. JSON-backed identity
  does not depend on editable text or array position.
- Before the first import with keys, legacy sections/modules are matched by
  parent and title, and vocabulary by module, reading, and written form. Apply
  this baseline catalog before renaming legacy content. Ambiguous matches abort
  rather than guessing which learner history to retain. Review unmatched old
  rows manually; they are not deleted.
- Different words with the same reading, and distinct senses with separate
  item IDs, remain separate items. The old importer collapsed these by reading.
- Built-in Go content receives deterministic keys on import. Its key generation
  still depends on its original parent/text; assign explicit keys before
  renaming built-in content. JSON catalog IDs are the preferred editorial model.
- Section orders 1–6 are reserved for built-in basics. Module orders must be
  positive and unique within each section. Changing order can change the
  predecessor required for unlocking, even though stored scores survive.
- Every item has the same fields: `id`, `kanji`, `kana`, `romaji`,
  `english_meaning`, `indonesian_meaning`, `part_of_speech`, `example_sentence`,
  and `source`. A reading and at least one meaning are required. Empty optional
  strings are allowed; blank romaji is derived from kana.
- `source` records the original local dataset path. It is provenance, not proof
  of a license or editorial accuracy. Translation accuracy still needs human
  review. The importer does not invent translations.
- Unknown fields, invalid IDs, duplicate orders/IDs, invalid pass thresholds,
  missing readings/meanings, corrupted text, and database-length violations fail
  validation before any database connection.

## Conversion audit

The published catalog contains 3 JSON-backed sections, 246 topic modules, and
5,663 entries. The former Daily Vocab section and the 115 source-oriented
`Day X` / `Page X` modules are registered as retired. They remain in the
database, hidden, so existing sessions, scores, and review history keep valid
foreign keys.

Daily Vocab is now treated as source data. Its Indonesian translations are
merged into matching N3 entries, and its unique words are distributed into N3
topic modules. Raw N5, N4, and N3 page entries are likewise distributed into
stable topic modules such as Work & Business, Travel & Transport, Health &
Body, and Grammar, Adverbs & Expressions. Curated modules keep their stable IDs
and use the `Core — topic` title prefix; redistributed source material uses
`Extended — topic`.

Transitive/intransitive pairs in day files are flattened into individual entries
with distinct stable IDs and part-of-speech labels. The old loader skipped them.
One malformed N3 Christmas entry has a missing reading and replacement
characters. Its original content is preserved in `data/catalog/quarantine.json`
and in the raw source, but is excluded from new imports pending editorial review.
If it already exists in a database, it is left untouched.

`convert-legacy.cjs` documents the initial transformation and refuses to
overwrite an existing catalog. `reorganize-curriculum.cjs` documents this
topic-based publishing transformation and defaults to a dry run; pass `--write`
only when intentionally updating the canonical catalog. Do not regenerate IDs
from source positions when editing a published catalog. The unused
`data/json/kotoba` and `data/json/n5` reference datasets remain unused, as
before.

Run regression tests with `go test ./cmd/seed`. They cover full-catalog repeat
imports, legacy-ID adoption, preservation of scores/reviews, homophones, text
edits, removed source entries, validation failures, ambiguity, and rollback.
