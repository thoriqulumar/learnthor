// One-time conversion of the original sources. Never overwrite the edited catalog.
const fs = require('node:fs');
const path = require('node:path');
const root = path.join(__dirname, 'data');
const destination = path.join(root, 'catalog', 'curriculum.json');
if (fs.existsSync(destination)) throw new Error('Catalog already exists; edit it directly to preserve IDs.');
const sections = new Map();
const quarantined = [];
const read = file => JSON.parse(fs.readFileSync(path.join(root, file), 'utf8'));
const clean = value => (value || '').normalize('NFC').trim();
function section(id, title, order, premium) {
  if (!sections.has(id)) sections.set(id, { id, title, sort_order: order, is_premium: premium, modules: [] });
  return sections.get(id);
}
function item(row, id, source, indonesian = false) {
  const result = {
    id, kanji: clean(row.kanji), kana: clean(row.kana || row.reading || row.hiragana),
    romaji: clean(row.romaji), english_meaning: clean(indonesian ? '' : row.meaning_en || row.meaning),
    indonesian_meaning: clean(indonesian ? row.meaning : row.meaning_id),
    part_of_speech: clean(row.pos), example_sentence: clean(row.example), source,
  };
  if (!result.kana || Object.values(result).some(v => v.includes('\uFFFD'))) {
    quarantined.push({ id, source, reason: 'Missing reading or replacement character; needs editorial review', original: row });
    return null;
  }
  return result;
}
for (const file of fs.readdirSync(root).filter(f => f.endsWith('.json')).sort()) {
  for (const s of read(file)) {
    const level = s.section.match(/N[345]/)[0].toLowerCase();
    const target = section(level, s.section, s.sort_order, level !== 'n5');
    s.modules.forEach((m, index) => {
      const id = `curated-${level}-${file.replace('.json', '').replaceAll('_', '-')}-${index + 1}`;
      target.modules.push({ id, title: m.title, sort_order: m.sort_order, pass_threshold: m.pass_threshold,
        items: m.items.map((r, i) => item(r, `${id}-${i + 1}`, file)).filter(Boolean) });
    });
  }
}
for (const level of ['n5', 'n4', 'n3']) {
  const folder = `json/vocab/vocab/${level}`;
  for (const file of fs.readdirSync(path.join(root, folder)).filter(f => /^page_\d+\.json$/.test(f)).sort((a, b) => Number(a.match(/\d+/)[0]) - Number(b.match(/\d+/)[0]))) {
    const page = Number(file.match(/\d+/)[0]);
    const id = `${level}-page-${page}`;
    sections.get(level).modules.push({ id, title: `${level.toUpperCase()} Vocab — Page ${page}`,
      sort_order: 99 + page, pass_threshold: 90,
      items: read(`${folder}/${file}`).map((r, i) => item(r, `${id}-${i + 1}`, `${folder}/${file}`)).filter(Boolean) });
  }
}
const daily = section('daily', 'Daily Vocab (Day 29–59)', 100, true);
for (const file of fs.readdirSync(path.join(root, 'json')).filter(f => /^day-\d+\.json$/.test(f)).sort()) {
  const day = Number(file.match(/\d+/)[0]);
  const id = `day-${day}`;
  const rows = [];
  read(`json/${file}`).forEach((r, i) => {
    if (r.transitive || r.intransitive) {
      for (const type of ['transitive', 'intransitive']) {
        if (r[type]) rows.push(item({ ...r[type], pos: `${type} verb` }, `${id}-${i + 1}-${type}`, `json/${file}`, true));
      }
    } else rows.push(item(r, `${id}-${i + 1}`, `json/${file}`, true));
  });
  daily.modules.push({ id, title: `Day ${day}`, sort_order: day, pass_threshold: 90, items: rows.filter(Boolean) });
}
const catalog = { schema_version: 1, sections: [...sections.values()].sort((a, b) => a.sort_order - b.sort_order) };
fs.mkdirSync(path.dirname(destination), { recursive: true });
fs.writeFileSync(destination, JSON.stringify(catalog, null, 2) + '\n');
fs.writeFileSync(path.join(root, 'catalog', 'quarantine.json'), JSON.stringify(quarantined, null, 2) + '\n');
console.log(JSON.stringify({ sections: catalog.sections.length, modules: catalog.sections.flatMap(s => s.modules).length, items: catalog.sections.flatMap(s => s.modules).flatMap(m => m.items).length, quarantined: quarantined.length }));
