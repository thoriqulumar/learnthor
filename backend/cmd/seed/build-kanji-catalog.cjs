const fs = require('node:fs');
const path = require('node:path');

const root = __dirname;
const sourceDir = path.join(root, 'data', 'kanji-source');
const outputPath = path.join(root, 'data', 'catalog', 'kanji.json');
const curriculumPath = path.join(root, 'data', 'catalog', 'curriculum.json');
const translationsPath = path.join(root, 'data', 'kanji-source', 'translations-id.json');
const svgDir = path.resolve(root, '..', '..', '..', 'frontend', 'public', 'kanji');
const translate = process.argv.includes('--translate');
const downloadSVG = process.argv.includes('--download-svg');

const curriculum = JSON.parse(fs.readFileSync(curriculumPath, 'utf8'));
const knownIndonesian = new Map();
for (const section of curriculum.sections) {
  for (const module of section.modules) {
    for (const item of module.items) {
      if (item.kanji && [...item.kanji].length === 1 && item.indonesian_meaning) {
        knownIndonesian.set(item.kanji, item.indonesian_meaning);
      }
    }
  }
}

const existing = fs.existsSync(outputPath)
  ? new Map(JSON.parse(fs.readFileSync(outputPath, 'utf8')).entries.map((entry) => [entry.character, entry.indonesian_meaning]))
  : new Map();
const machineTranslations = fs.existsSync(translationsPath)
  ? new Map(Object.entries(JSON.parse(fs.readFileSync(translationsPath, 'utf8'))))
  : new Map();

const entries = ['n5', 'n4', 'n3'].flatMap((level) => {
  const rows = JSON.parse(fs.readFileSync(path.join(sourceDir, `${level}.json`), 'utf8'));
  return rows.map((row, index) => ({
    id: `kanji-${level}-${row.character.codePointAt(0).toString(16)}`,
    character: row.character,
    jlpt_level: row.level,
    sort_order: index + 1,
    stroke_count: row.strokes,
    grade: row.grade || 0,
    frequency: row.freq || 0,
    radical: row.radical || '',
    on_readings: row.onyomi || [],
    kun_readings: row.kunyomi || [],
    english_meaning: (row.meanings || []).join(', '),
    indonesian_meaning: machineTranslations.get(row.character) || existing.get(row.character) || knownIndonesian.get(row.character) || '',
    example_words: (row.words || []).slice(0, 8),
    stroke_svg: `/kanji/${row.character.codePointAt(0).toString(16).padStart(5, '0')}.svg`,
  }));
});

const delay = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

async function translateBatch(batch) {
  const source = batch.map((entry) => `@@${entry.character.codePointAt(0).toString(16)}@@ ${entry.english_meaning}`).join('\n');
  const url = new URL('https://translate.googleapis.com/translate_a/single');
  url.searchParams.set('client', 'gtx');
  url.searchParams.set('sl', 'en');
  url.searchParams.set('tl', 'id');
  url.searchParams.set('dt', 't');
  url.searchParams.set('q', source);
  let response;
  for (let attempt = 1; attempt <= 5; attempt++) {
    response = await fetch(url, { signal: AbortSignal.timeout(30000) });
    if (response.status !== 429) break;
    await delay(attempt * 3000);
  }
  if (!response.ok) throw new Error(`translation batch failed: ${response.status}`);
  const body = await response.json();
  const translated = body[0].map((part) => part[0]).join('');
  const matches = [...translated.matchAll(/@@([0-9a-f]+)@@\s*([^@]+?)(?=\n@@|$)/gi)];
  const byCode = new Map(matches.map((match) => [match[1].toLowerCase(), match[2].trim()]));
  for (const entry of batch) {
    entry.indonesian_meaning = byCode.get(entry.character.codePointAt(0).toString(16)) || '';
  }
}

async function translateMissing() {
  if (!translate) return;
  const missing = entries.filter((entry) => !entry.indonesian_meaning);
  for (let index = 0; index < missing.length; index += 20) {
    await translateBatch(missing.slice(index, index + 20));
    await delay(400);
  }
}

async function downloadDiagram(entry) {
  if (!downloadSVG) return;
  fs.mkdirSync(svgDir, { recursive: true });
  const destination = path.join(svgDir, path.basename(entry.stroke_svg));
  if (fs.existsSync(destination)) return;
  const source = `https://raw.githubusercontent.com/KanjiVG/kanjivg/master/kanji/${path.basename(entry.stroke_svg)}`;
  const response = await fetch(source, { signal: AbortSignal.timeout(15000) });
  if (!response.ok) throw new Error(`stroke diagram failed for ${entry.character}: ${response.status}`);
  fs.writeFileSync(destination, Buffer.from(await response.arrayBuffer()));
}

async function runPool(action, size = 8) {
  let next = 0;
  async function worker() {
    while (next < entries.length) {
      const entry = entries[next++];
      await action(entry);
    }
  }
  await Promise.all(Array.from({ length: size }, worker));
}

(async () => {
  await translateMissing();
  await runPool(downloadDiagram, 12);
  const missing = entries.filter((entry) => !entry.indonesian_meaning);
  if (translate && missing.length) throw new Error(`${missing.length} Indonesian meanings are missing`);
  const catalog = {
    schema_version: 1,
    source: 'OpenJLPT / KANJIDIC2; Indonesian glosses augmented from Learnthor and machine translation',
    entries,
  };
  fs.writeFileSync(outputPath, `${JSON.stringify(catalog, null, 2)}\n`);
  console.log(JSON.stringify({ entries: entries.length, missing_indonesian: missing.length, svg_directory: svgDir }, null, 2));
})().catch((error) => {
  console.error(error);
  process.exit(1);
});
