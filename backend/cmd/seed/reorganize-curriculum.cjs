const fs = require('node:fs');
const path = require('node:path');

const catalogPath = path.join(__dirname, 'data', 'catalog', 'curriculum.json');
const write = process.argv.includes('--write');
const catalog = JSON.parse(fs.readFileSync(catalogPath, 'utf8'));

function normalizePublishedTitles(value) {
  let renamed = 0;
  for (const section of value.sections) {
    for (const module of section.modules) {
      const kind = module.id.startsWith('organized-') ? 'Extended' : 'Core';
      const topic = module.title
        .replace(/^(?:Core|Extended) — /, '')
        .replace(/^N[345]:\s*/, '');
      const title = `${kind} — ${topic}`;
      if (module.title !== title) {
        module.title = title;
        renamed++;
      }
    }
  }
  return renamed;
}

if (!catalog.sections.some((section) => section.id === 'daily')) {
  if (!(catalog.retired_sections || []).includes('daily')) {
    throw new Error('Daily section is missing and is not registered as retired');
  }
  const modules = catalog.sections.flatMap((section) => section.modules);
  const renamed = normalizePublishedTitles(catalog);
  if (write && renamed > 0) fs.writeFileSync(catalogPath, `${JSON.stringify(catalog, null, 2)}\n`);
  console.log(JSON.stringify({
    mode: write ? (renamed > 0 ? 'written' : 'unchanged') : 'dry-run',
    status: 'already reorganized',
    renamed,
    sections: catalog.sections.length,
    modules: modules.length,
    items: modules.flatMap((module) => module.items).length,
  }, null, 2));
  process.exit(0);
}

const normalize = (value) => (value || '').normalize('NFKC').trim().toLowerCase();
const wordKey = (item) => `${normalize(item.kanji)}|${normalize(item.kana)}`;
const roman = (number) => {
  const values = [[10, 'X'], [9, 'IX'], [5, 'V'], [4, 'IV'], [1, 'I']];
  let result = '';
  for (const [value, numeral] of values) {
    while (number >= value) {
      result += numeral;
      number -= value;
    }
  }
  return result;
};

const categories = [
  ['people-relationships', 'People & Relationships', ['person', 'people', 'family', 'friend', 'parent', 'mother', 'father', 'brother', 'sister', 'child', 'husband', 'wife', 'relationship', 'marriage', 'guest', 'customer', 'orang', 'keluarga', 'teman', 'ibu', 'ayah', 'saudara', 'hubungan', 'pernikahan', 'pelanggan']],
  ['home-daily-life', 'Home & Daily Life', ['house', 'home', 'room', 'furniture', 'clothes', 'clothing', 'daily', 'household', 'kitchen', 'bath', 'clean', 'wash', 'tidy', 'rumah', 'kamar', 'pakaian', 'sehari', 'dapur', 'mandi', 'membersihkan', 'mencuci']],
  ['work-business', 'Work & Business', ['work', 'job', 'office', 'company', 'business', 'career', 'salary', 'meeting', 'industry', 'production', 'factory', 'manager', 'employee', 'kerja', 'kantor', 'perusahaan', 'bisnis', 'gaji', 'rapat', 'industri', 'produksi', 'pabrik', 'lembur']],
  ['school-learning', 'School & Learning', ['school', 'student', 'teacher', 'study', 'learn', 'lesson', 'class', 'exam', 'university', 'education', 'book', 'sekolah', 'murid', 'guru', 'belajar', 'pelajaran', 'kelas', 'ujian', 'pendidikan', 'buku']],
  ['society-government', 'Society, Law & Government', ['society', 'social', 'government', 'politic', 'law', 'rule', 'public', 'population', 'war', 'peace', 'international', 'country', 'masyarakat', 'sosial', 'pemerintah', 'politik', 'hukum', 'aturan', 'publik', 'penduduk', 'perang', 'negara']],
  ['money-economy', 'Money & Economy', ['money', 'price', 'cost', 'econom', 'bank', 'tax', 'pay', 'payment', 'buy', 'sell', 'shop', 'market', 'uang', 'harga', 'biaya', 'ekonomi', 'pajak', 'membayar', 'membeli', 'menjual', 'pasar']],
  ['travel-transport', 'Travel & Transport', ['travel', 'trip', 'train', 'car', 'bus', 'station', 'road', 'street', 'ticket', 'airport', 'ship', 'arrive', 'depart', 'perjalanan', 'kereta', 'mobil', 'stasiun', 'jalan', 'tiket', 'bandara', 'kapal', 'berangkat']],
  ['health-body', 'Health & Body', ['health', 'medical', 'medicine', 'doctor', 'hospital', 'body', 'pain', 'ill', 'sick', 'injury', 'fever', 'blood', 'bone', 'heart', 'kesehatan', 'medis', 'obat', 'dokter', 'rumah sakit', 'tubuh', 'sakit', 'cedera', 'demam', 'darah', 'tulang']],
  ['food-cooking', 'Food & Cooking', ['food', 'meal', 'cook', 'restaurant', 'eat', 'drink', 'rice', 'meat', 'fish', 'vegetable', 'fruit', 'taste', 'makanan', 'memasak', 'restoran', 'makan', 'minum', 'nasi', 'daging', 'ikan', 'sayur', 'buah', 'rasa']],
  ['nature-environment', 'Nature & Environment', ['nature', 'weather', 'season', 'rain', 'snow', 'wind', 'river', 'mountain', 'sea', 'ocean', 'tree', 'plant', 'animal', 'earth', 'environment', 'alam', 'cuaca', 'musim', 'hujan', 'salju', 'angin', 'sungai', 'gunung', 'laut', 'pohon', 'hewan', 'lingkungan']],
  ['technology-media', 'Technology & Media', ['technology', 'computer', 'internet', 'phone', 'email', 'machine', 'science', 'media', 'news', 'broadcast', 'screen', 'teknologi', 'komputer', 'telepon', 'mesin', 'sains', 'berita', 'layar']],
  ['culture-leisure', 'Culture, Arts & Leisure', ['culture', 'art', 'music', 'movie', 'film', 'sport', 'game', 'hobby', 'festival', 'tradition', 'literature', 'play', 'budaya', 'seni', 'musik', 'olahraga', 'permainan', 'hobi', 'tradisi', 'sastra']],
  ['time-quantity', 'Time, Frequency & Quantity', ['time', 'day', 'week', 'month', 'year', 'hour', 'minute', 'second', 'before', 'after', 'recent', 'often', 'sometimes', 'always', 'amount', 'number', 'waktu', 'hari', 'minggu', 'bulan', 'tahun', 'jam', 'menit', 'detik', 'sering', 'kadang', 'jumlah']],
  ['communication-language', 'Communication & Language', ['speak', 'say', 'tell', 'talk', 'ask', 'answer', 'reply', 'message', 'letter', 'language', 'word', 'meaning', 'explain', 'report', 'bicara', 'mengatakan', 'bertanya', 'jawab', 'balasan', 'pesan', 'bahasa', 'kata', 'arti', 'menjelaskan', 'laporan']],
  ['emotions-mind', 'Emotions, Mind & Perception', ['feel', 'emotion', 'mind', 'think', 'believe', 'remember', 'forget', 'happy', 'sad', 'angry', 'fear', 'love', 'hate', 'surprise', 'perasaan', 'pikiran', 'berpikir', 'percaya', 'ingat', 'lupa', 'senang', 'sedih', 'marah', 'takut', 'cinta', 'terkejut']],
  ['actions-change', 'Actions, Movement & Change', ['to ', 'action', 'move', 'change', 'begin', 'start', 'finish', 'stop', 'open', 'close', 'put', 'take', 'make', 'use', 'increase', 'decrease', 'become', 'melakukan', 'bergerak', 'berubah', 'memulai', 'selesai', 'berhenti', 'membuka', 'menutup', 'menggunakan', 'menambah', 'menurun', 'menjadi']],
  ['descriptions-states', 'Descriptions & States', ['state', 'condition', 'appearance', 'beautiful', 'large', 'small', 'high', 'low', 'long', 'short', 'new', 'old', 'strong', 'weak', 'easy', 'difficult', 'keadaan', 'kondisi', 'indah', 'besar', 'kecil', 'tinggi', 'rendah', 'panjang', 'pendek', 'baru', 'lama', 'kuat', 'lemah', 'mudah', 'sulit']],
  ['grammar-expressions', 'Grammar, Adverbs & Expressions', ['grammar', 'expression', 'phrase', 'usually', 'probably', 'perhaps', 'however', 'therefore', 'although', 'almost', 'enough', 'very', 'quite', 'not at all', 'tata bahasa', 'ungkapan', 'biasanya', 'mungkin', 'tetapi', 'karena itu', 'hampir', 'cukup', 'sangat', 'tidak sama sekali']],
  ['general', 'General Vocabulary', []],
];

function classify(item) {
  const text = normalize(`${item.english_meaning} ${item.indonesian_meaning} ${item.part_of_speech}`);
  let best = categories[categories.length - 1];
  let bestScore = 0;
  for (const category of categories.slice(0, -1)) {
    let score = 0;
    for (const keyword of category[2]) {
      if (text.includes(keyword)) score += keyword.includes(' ') ? 3 : 1;
    }
    if (score > bestScore) {
      best = category;
      bestScore = score;
    }
  }
  return best;
}

function mergeItem(target, incoming) {
  for (const field of ['kanji', 'romaji', 'english_meaning', 'part_of_speech', 'example_sentence']) {
    if (!target[field] && incoming[field]) target[field] = incoming[field];
  }
  const incomingID = (incoming.indonesian_meaning || '').trim();
  const currentID = (target.indonesian_meaning || '').trim();
  if (!currentID && incomingID) target.indonesian_meaning = incomingID;
  else if (incomingID && currentID !== incomingID && !currentID.includes(incomingID)) {
    const combined = `${currentID}; ${incomingID}`;
    if ([...combined].length <= 500) target.indonesian_meaning = combined;
  }
}

const retiredSections = new Set(catalog.retired_sections || []);
const retiredModules = new Set(catalog.retired_modules || []);
const stats = { retiredSections: 0, retiredModules: 0, merged: 0, organized: 0, omittedDuplicates: 0, publishedModules: 0 };
const distribution = {};
const levelState = new Map();

for (const level of ['n5', 'n4', 'n3']) {
  const section = catalog.sections.find((candidate) => candidate.id === level);
  if (!section) throw new Error(`Missing ${level} section`);
  const sourceModules = section.modules.filter((module) => module.id.startsWith(`${level}-page-`));
  const curatedModules = section.modules.filter((module) => !module.id.startsWith(`${level}-page-`) && !module.id.startsWith(`organized-${level}-`));
  for (const module of sourceModules) retiredModules.add(module.id);
  stats.retiredModules += sourceModules.length;

  const exact = new Map();
  const byKana = new Map();
  for (const module of curatedModules) {
    for (const item of module.items) {
      exact.set(wordKey(item), item);
      const kana = normalize(item.kana);
      if (!byKana.has(kana)) byKana.set(kana, []);
      byKana.get(kana).push(item);
    }
  }

  const pool = [];
  for (const module of sourceModules) {
    for (const item of module.items) {
      const existing = exact.get(wordKey(item));
      if (existing) {
        mergeItem(existing, item);
        stats.omittedDuplicates++;
        continue;
      }
      exact.set(wordKey(item), item);
      const kana = normalize(item.kana);
      if (!byKana.has(kana)) byKana.set(kana, []);
      byKana.get(kana).push(item);
      pool.push(item);
    }
  }
  levelState.set(level, { section, curatedModules, pool, exact, byKana });
}

const daily = catalog.sections.find((section) => section.id === 'daily');
retiredSections.add(daily.id);
stats.retiredSections++;
for (const module of daily.modules) retiredModules.add(module.id);
stats.retiredModules += daily.modules.length;

const n3 = levelState.get('n3');
for (const module of daily.modules) {
  for (const item of module.items) {
    let existing = n3.exact.get(wordKey(item));
    if (!existing) {
      const sameReading = n3.byKana.get(normalize(item.kana)) || [];
      if (sameReading.length === 1 && (!normalize(item.kanji) || !normalize(sameReading[0].kanji))) {
        existing = sameReading[0];
      }
    }
    if (existing) {
      mergeItem(existing, item);
      stats.merged++;
      continue;
    }
    n3.exact.set(wordKey(item), item);
    const kana = normalize(item.kana);
    if (!n3.byKana.has(kana)) n3.byKana.set(kana, []);
    n3.byKana.get(kana).push(item);
    n3.pool.push(item);
  }
}

for (const [level, state] of levelState) {
  const groups = new Map(categories.map(([slug, title]) => [slug, { slug, title, items: [] }]));
  for (const item of state.pool) groups.get(classify(item)[0]).items.push(item);
  distribution[level] = Object.fromEntries([...groups.values()].filter((group) => group.items.length > 0).map((group) => [group.title, group.items.length]));
  let sortOrder = Math.max(...state.curatedModules.map((module) => module.sort_order));
  const organized = [];
  for (const { slug, title, items } of groups.values()) {
    items.sort((a, b) => normalize(a.kana).localeCompare(normalize(b.kana), 'ja'));
    const parts = Math.ceil(items.length / 40);
    for (let index = 0; index < parts; index++) {
      const chunk = items.slice(index * 40, (index + 1) * 40);
      if (chunk.length === 0) continue;
      const suffix = parts > 1 ? ` ${roman(index + 1)}` : '';
      organized.push({
        id: `organized-${level}-${slug}-${index + 1}`,
        title: `${title}${suffix}`,
        sort_order: ++sortOrder,
        pass_threshold: 90,
        items: chunk,
      });
      stats.publishedModules++;
      stats.organized += chunk.length;
    }
  }
  state.section.modules = [...state.curatedModules, ...organized];
}

catalog.sections = catalog.sections.filter((section) => section.id !== 'daily');
catalog.retired_sections = [...retiredSections].sort();
catalog.retired_modules = [...retiredModules].sort();
stats.renamed = normalizePublishedTitles(catalog);

const output = `${JSON.stringify(catalog, null, 2)}\n`;
if (write) fs.writeFileSync(catalogPath, output);
console.log(JSON.stringify({ mode: write ? 'written' : 'dry-run', ...stats, sections: catalog.sections.length, modules: catalog.sections.reduce((count, section) => count + section.modules.length, 0), items: catalog.sections.flatMap((section) => section.modules).flatMap((module) => module.items).length, distribution }, null, 2));
