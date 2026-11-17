const fs = require('node:fs');
const path = require('node:path');
const file = path.join(__dirname, 'data/catalog/jlpt-materials.json');
const catalog = JSON.parse(fs.readFileSync(file, 'utf8'));
// IDs below are editorial identities; retain them when revising published content.
// Fields: level, id, form, rule EN, rule ID, example, example EN, example ID, distractors.
const entries = [
['N5','topic','は','Introduces the topic or a contrast. It is pronounced wa as a particle.','Mengenalkan topik atau perbandingan. Sebagai partikel dibaca wa.','私は学生です。','I am a student.','Saya seorang pelajar.',['を','に','で']],
['N5','subject','が','Identifies the subject, including newly introduced information with existence verbs.','Menandai subjek, termasuk informasi baru dengan kata kerja keberadaan.','庭に猫がいます。','There is a cat in the garden.','Ada seekor kucing di taman.',['を','で','へ']],
['N5','object','を','Marks a direct object. It also marks a route or a place left behind. Pronounced o.','Menandai objek langsung, jalur yang dilalui, atau tempat yang ditinggalkan. Dibaca o.','毎日、水を飲みます。','I drink water every day.','Saya minum air setiap hari.',['が','に','へ']],
['N5','ni','に','Marks a specific time, destination, recipient, or location of existence.','Menandai waktu tertentu, tujuan, penerima, atau tempat keberadaan.','七時に起きます。','I wake up at seven.','Saya bangun pukul tujuh.',['で','を','と']],
['N5','de','で','Marks an action location, means, material, or cause. Compare existence に with action で.','Menandai tempat aktivitas, sarana, bahan, atau sebab. Bandingkan keberadaan に dengan aktivitas で.','図書館で勉強します。','I study at the library.','Saya belajar di perpustakaan.',['に','へ','を']],
['N5','direction','へ','Marks direction toward a destination. It is pronounced e as a particle.','Menandai arah menuju tujuan. Sebagai partikel dibaca e.','日本へ行きます。','I go to Japan.','Saya pergi ke Jepang.',['で','を','と']],
['N5','possession','の','Connects nouns: possession, affiliation, description, or a noun standing for an understood item.','Menghubungkan kata benda: kepemilikan, afiliasi, penjelas, atau pengganti benda yang sudah dipahami.','これは私の本です。','This is my book.','Ini buku saya.',['を','で','へ']],
['N5','to','と','Lists all named items, marks a companion, or introduces quoted speech.','Mendaftar semua unsur yang disebutkan, menandai teman melakukan aktivitas, atau kutipan.','友達と話します。','I talk with a friend.','Saya berbicara dengan teman.',['を','へ','から']],
['N5','also','も','Means also or too. It replaces は, が, or を, but can follow に or で.','Berarti juga. Menggantikan は, が, atau を, tetapi dapat mengikuti に atau で.','田中さんも学生です。','Tanaka is also a student.','Tanaka juga seorang pelajar.',['を','に','で']],
['N5','examples','や','Lists representative examples rather than a complete list; often paired with など.','Mendaftar contoh yang tidak lengkap; sering berpasangan dengan など.','パンや果物などを買います。','I buy things such as bread and fruit.','Saya membeli makanan seperti roti dan buah.',['を','で','へ']],
['N5','from','から','Marks a starting point in time or space; after a clause, gives a reason.','Menandai titik awal waktu atau tempat; setelah klausa, menyatakan alasan.','授業は九時からです。','Class starts at nine.','Kelas dimulai pukul sembilan.',['まで','を','で']],
['N5','until','まで','Marks the endpoint of a range or duration. Compare a deadline with までに.','Menandai akhir rentang atau durasi. Bandingkan batas waktu penyelesaian dengan までに.','五時まで働きます。','I work until five.','Saya bekerja sampai pukul lima.',['から','を','と']],
['N5','question','か','Forms a polite question or presents alternatives with AかB.','Membentuk pertanyaan sopan atau menyatakan alternatif dengan AかB.','これは何ですか。','What is this?','Apa ini?',['を','に','で']],
['N5','shared','ね','Seeks agreement or expresses shared feeling or information.','Meminta persetujuan atau menyatakan perasaan maupun informasi bersama.','いい天気ですね。','Nice weather, isn’t it?','Cuacanya bagus, ya?',['を','へ','から']],
['N5','inform','よ','Informs or emphasizes information that the listener may not know.','Memberi atau menegaskan informasi yang mungkin belum diketahui pendengar.','この店は安いですよ。','This shop is inexpensive, you know.','Toko ini murah, lho.',['を','へ','まで']],
['N5','only','だけ','Means only without requiring a negative verb.','Berarti hanya dan tidak membutuhkan kata kerja negatif.','水だけ飲みます。','I drink only water.','Saya hanya minum air.',['へ','に','から']],
['N5','approximately','くらい','Means approximately for an amount or duration; ぐらい is a common variant.','Berarti kira-kira untuk jumlah atau durasi; ぐらい merupakan variasi umum.','三十分くらい待ちました。','I waited about thirty minutes.','Saya menunggu sekitar tiga puluh menit.',['へ','を','から']],
['N5','comparison','より','Marks the reference point in a comparison: AはBより...','Menandai pembanding: AはBより...','電車はバスより速いです。','The train is faster than the bus.','Kereta lebih cepat daripada bus.',['を','に','で']],
['N4','deadline','までに','Sets a deadline for completing an action; まで describes continuing until an endpoint.','Menetapkan batas penyelesaian aktivitas; まで menyatakan aktivitas berlanjut sampai titik akhir.','金曜日までに出してください。','Please submit it by Friday.','Harap serahkan paling lambat Jumat.',['から','ながら','のに']],
['N4','restriction','しか','Means only with a negative predicate; emphasizes that the amount is limited.','Berarti hanya dengan predikat negatif; menekankan jumlah yang terbatas.','百円しかありません。','I have only one hundred yen.','Saya hanya memiliki seratus yen.',['だけ','こそ','ながら']],
['N4','degree','ほど','Marks degree or extent; in a negative comparison, AはBほど...ない means A is not as ... as B.','Menandai tingkat; dalam perbandingan negatif, AはBほど...ない berarti A tidak se... B.','今日は昨日ほど寒くないです。','Today is not as cold as yesterday.','Hari ini tidak sedingin kemarin.',['までに','のに','ながら']],
['N4','listing','など','Means such as, and so on, or things like that; marks representative examples.','Berarti seperti atau dan sebagainya; menandai contoh yang mewakili.','本などを売っています。','They sell books and similar things.','Mereka menjual buku dan barang sejenis.',['へ','しか','ながら']],
['N4','demo','でも','After a noun, suggests an example such as tea or expresses even; sentence-initial でも means but.','Setelah kata benda, mengusulkan contoh seperti teh atau berarti bahkan; di awal kalimat berarti tetapi.','お茶でも飲みませんか。','Would you like to have some tea or something?','Mau minum teh atau semacamnya?',['しか','までに','ながら']],
['N4','reasons','し','Lists reasons or attributes, often implying that there are other reasons too.','Mendaftar alasan atau sifat dan sering menyiratkan masih ada alasan lain.','この店は安いし、おいしいです。','This shop is inexpensive, and the food is delicious.','Toko ini murah dan makanannya enak.',['ながら','までに','しか']],
['N4','conditional','と','After a non-past clause, expresses a regular or automatic result. Distinguish it from noun-listing と.','Setelah klausa nonlampau, menyatakan hasil teratur atau otomatis. Bedakan dari と untuk daftar kata benda.','このボタンを押すと、ドアが開きます。','When you press this button, the door opens.','Jika tombol ini ditekan, pintu terbuka.',['ながら','のに','までに']],
['N4','contrast','けれど','Connects contrasting clauses or softens a statement. けど and けれども are related forms.','Menghubungkan klausa berlawanan atau memperhalus pernyataan. けど dan けれども adalah bentuk terkait.','高いけれど、買いたいです。','It is expensive, but I want to buy it.','Harganya mahal, tetapi saya ingin membelinya.',['ながら','までに','しか']],
['N4','nominalizer','の','Turns a clause into a noun-like unit; in のが好き, the whole activity is what is liked.','Mengubah klausa menjadi unsur seperti kata benda; dalam のが好き, seluruh aktivitas menjadi hal yang disukai.','泳ぐのが好きです。','I like swimming.','Saya suka berenang.',['へ','しか','まで']],
['N4','quotation','って','A conversational quotation or topic marker; use と for more neutral quoted speech.','Penanda kutipan atau topik dalam percakapan; gunakan と untuk kutipan yang lebih netral.','彼は来ないって言いました。','He said that he would not come.','Dia mengatakan bahwa dia tidak akan datang.',['ながら','までに','しか']],
['N3','dake-de-naku','だけでなく','Means not only A but also B, commonly followed by も in the second part.','Berarti bukan hanya A tetapi juga B, sering diikuti も pada bagian kedua.','彼は英語だけでなく、日本語も話せます。','He can speak not only English but also Japanese.','Dia bisa berbicara bukan hanya bahasa Inggris tetapi juga bahasa Jepang.',['しか','までに','ながら']],
['N3','nado-focus','なんか','A conversational form related to など; can give examples or downplay the named item.','Bentuk percakapan terkait など; dapat memberikan contoh atau merendahkan pentingnya unsur yang disebut.','私なんかには無理です。','It is impossible for someone like me.','Itu tidak mungkin bagi orang seperti saya.',['までに','ながら','によって']],
['N3','nanka-nante','なんて','Introduces a quoted idea with surprise, evaluation, or dismissal; often means something like ... .','Memperkenalkan gagasan dengan rasa terkejut, penilaian, atau penolakan; sering berarti hal seperti ... .','一人で行くなんて、危ないです。','Going alone like that is dangerous.','Pergi sendirian seperti itu berbahaya.',['までに','を通して','に対して']],
['N3','wa-contrast','では','Combines で with contrastive は to frame a location, context, or means as the topic.','Menggabungkan で dengan は untuk menjadikan tempat, konteks, atau sarana sebagai topik perbandingan.','日本では左側を走ります。','In Japan, vehicles drive on the left.','Di Jepang, kendaraan berjalan di sisi kiri.',['を','へ','から']],
['N3','niwa','には','Combines に with は to contrast a destination, time, recipient, or context.','Menggabungkan に dengan は untuk membandingkan tujuan, waktu, penerima, atau konteks.','私には難しいです。','It is difficult for me.','Itu sulit bagi saya.',['を','へ','から']],
['N3','about','について','Means about or concerning a topic; についての modifies a following noun.','Berarti tentang atau mengenai topik; についての menerangkan kata benda berikutnya.','日本の歴史について調べます。','I research Japanese history.','Saya meneliti tentang sejarah Jepang.',['によって','に比べて','として']],
['N3','toward','に対して','Marks the target of an attitude or action; can also contrast two situations.','Menandai sasaran sikap atau tindakan; juga dapat membandingkan dua keadaan.','お客さんに対して丁寧に話します。','I speak politely to customers.','Saya berbicara dengan sopan kepada pelanggan.',['について','によって','に比べて']],
['N3','depending','によって','Expresses depending on, means, cause, or the agent of a passive action.','Menyatakan bergantung pada, sarana, sebab, atau pelaku dalam kalimat pasif.','国によって習慣が違います。','Customs differ depending on the country.','Kebiasaan berbeda tergantung negaranya.',['に対して','について','として']],
['N3','as','として','Marks a role, capacity, or status: as a teacher, as a representative, and so on.','Menandai peran, kapasitas, atau status: sebagai guru, sebagai wakil, dan sebagainya.','教師として働いています。','I work as a teacher.','Saya bekerja sebagai guru.',['について','によって','に比べて']],
['N3','compared','に比べて','Means compared with; the following clause states the difference.','Berarti dibandingkan dengan; klausa berikutnya menyatakan perbedaannya.','去年に比べて、暑いです。','It is hot compared with last year.','Cuacanya panas dibandingkan tahun lalu.',['について','として','を通して']],
['N3','through','を通して','Means through a medium, experience, or period; を通じて is a related form.','Berarti melalui sarana, pengalaman, atau periode; を通じて adalah bentuk terkait.','旅行を通して友達ができました。','I made friends through travel.','Saya mendapat teman melalui perjalanan.',['について','に対して','に比べて']],
['N3','according','によると','Introduces the source of reported information; commonly followed by そうだ or another reporting expression.','Memperkenalkan sumber informasi; sering diikuti そうだ atau ungkapan laporan lain.','天気予報によると、明日は雨だそうです。','According to the forecast, it will rain tomorrow.','Menurut prakiraan cuaca, besok katanya akan hujan.',['として','に比べて','に対して']],
['N3','opposition','にとって','Means for or from the standpoint of someone; introduces whose evaluation follows.','Berarti bagi atau dari sudut pandang seseorang; menunjukkan siapa yang memberikan penilaian.','私にとって、家族が大切です。','For me, family is important.','Bagi saya, keluarga itu penting.',['について','によると','を通して']],
['N3','even-made','まで','Besides an endpoint, can emphasize an unexpected extreme: even this person or thing.','Selain titik akhir, dapat menekankan contoh ekstrem yang tidak terduga: bahkan orang atau benda ini.','先生まで間違えました。','Even the teacher made a mistake.','Bahkan guru pun melakukan kesalahan.',['から','ながら','について']],
['N3','though-mono','ものの','Connects a fact to a contrasting outcome: although; generally more formal than けど.','Menghubungkan fakta dengan hasil yang berlawanan: meskipun; umumnya lebih formal daripada けど.','買ったものの、まだ使っていません。','Although I bought it, I have not used it yet.','Meskipun sudah membelinya, saya belum menggunakannya.',['ので','ながら','によって']],
['N3','while-nagara','ながらも','Expresses although or despite after a stem or descriptive expression; distinguish it from simultaneous ながら.','Menyatakan meskipun setelah akar atau ungkapan deskriptif; bedakan dari ながら untuk aktivitas bersamaan.','小さいながらも、便利な家です。','Although it is small, it is a convenient house.','Meskipun kecil, rumah ini nyaman digunakan.',['までに','によって','について']],
];

for (const [level,key,form,en,id,jp,translationEn,translationId,distractors] of entries) {
  const lessonId = level.toLowerCase() + '-particle-' + key;
  const existing = catalog.lessons.find(l => l.id === lessonId);
  const levelIndex = entries.filter(e => e[0] === level).findIndex(e => e[1] === key);
  const options = [...distractors];
  const correctIndex = levelIndex % 4;
  options.splice(correctIndex, 0, form);
  const formPosition = jp.lastIndexOf(form);
  if (formPosition < 0) throw new Error('Missing particle in ' + lessonId);
  const blank = jp.slice(0,formPosition) + '＿' + jp.slice(formPosition + form.length);
  const otherEntries = entries.filter(e => e[0] === level && e[1] !== key).slice(0,3);
  const question = (suffix,promptEn,promptId,choices,index) => ({
    id: lessonId + '-' + suffix, prompt_en: promptEn, prompt_id: promptId,
    options: choices, correct_index: index, explanation_en: en, explanation_id: id,
  });
  const lesson = {
    id: lessonId, level, category: 'particles', sort_order: levelIndex + 2,
    is_premium: level !== 'N5', title_en: form + ' — usage and practice', title_id: form + ' — penggunaan dan latihan',
    summary_en: en, summary_id: id,
    content: [
      {heading_en: 'How to use ' + form, heading_id: 'Cara menggunakan ' + form, body_en: en, body_id: id,
       examples: [{japanese: jp, english: translationEn, indonesian: translationId}]},
      {heading_en: 'Read the particle in context', heading_id: 'Baca partikel dalam konteks',
       body_en: 'Identify the word or clause before ' + form + ', then the relationship described by the rule above. Higher-level lessons reuse particles from earlier levels. Level placement is a teaching guide, not an official exhaustive JLPT list.',
       body_id: 'Kenali kata atau klausa sebelum ' + form + ', lalu hubungan yang dijelaskan pada aturan di atas. Pelajaran tingkat lanjut memakai kembali partikel dari tingkat sebelumnya. Penempatan level adalah panduan belajar, bukan daftar resmi JLPT yang lengkap.',
       japanese_text: jp}
    ],
    questions: [
      question('context','Complete the sentence: ' + blank,'Lengkapi kalimat: ' + blank,options,correctIndex),
      question('function','Which rule explains ' + form + ' in: ' + jp,'Aturan mana yang menjelaskan ' + form + ' dalam: ' + jp,
        [en + ' / ' + id,...otherEntries.map(e=>e[3]+' / '+e[4])],0),
      question('meaning','Choose the meaning of: ' + jp,'Pilih arti: ' + jp,
        [translationEn + ' / ' + translationId,...otherEntries.map(e=>e[6]+' / '+e[7])],0)
    ]
  };
  if (existing) Object.assign(existing,lesson); else catalog.lessons.push(lesson);
}
fs.writeFileSync(file,JSON.stringify(catalog,null,2)+'\n','utf8');
const particles=catalog.lessons.filter(l=>l.category==='particles');
console.log(JSON.stringify({lessons:catalog.lessons.length,particle_lessons:particles.length,particle_questions:particles.reduce((s,l)=>s+l.questions.length,0)}));
