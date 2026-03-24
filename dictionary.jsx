import { useState } from "react";

/* ─────────── DATA ─────────── */
const WORDS = [
  {
    id: "1", word: "ambiguous", pos: "adj", ipa: "/æmˈbɪɡ.ju.əs/",
    translation: "двусмысленный",
    topics: ["Advanced Vocabulary"],
    senses: [
      { pos: "Adjective", def: "Open to more than one interpretation; not having one obvious meaning", trans: "двусмысленный, неоднозначный", examples: [{ en: "The ending of the film was deliberately ambiguous.", ru: "Концовка фильма была намеренно двусмысленной." }] }
    ],
    created: "2026-03-05",
    srs: { level: 3, next: "2026-03-25", streak: 5, history: [true, true, false, true, true] },
    notes: "Often confused with 'vague'. Ambiguous = multiple meanings, vague = unclear."
  },
  {
    id: "2", word: "audacity", pos: "n", ipa: "/ɔːˈdæs.ə.ti/",
    translation: "смелость, дерзость",
    topics: ["Advanced Vocabulary", "Emotions & Feelings"],
    senses: [
      { pos: "Noun", def: "A willingness to take bold risks", trans: "смелость, дерзость", examples: [{ en: "She had the audacity to challenge the CEO's decision.", ru: "У неё хватило дерзости оспорить решение генерального директора." }] },
      { pos: "Noun", def: "Rude or disrespectful behaviour; impudence", trans: "наглость", examples: [{ en: "He had the audacity to show up uninvited.", ru: "У него хватило наглости прийти без приглашения." }] }
    ],
    encounters: [
      { type: "book", icon: "📚", title: "The Audacity of Hope", author: "Barack Obama", detail: "Book title — positive sense", date: "2026-02-14" },
      { type: "tv", icon: "🎬", title: "The Office", author: "S03E07", detail: "\"The audacity!\" — Michael Scott", date: "2026-02-20" },
      { type: "web", icon: "🌐", title: "The Economist", author: "Article", detail: "\"entrepreneurial audacity\" — about startup founders", date: "2026-03-01" },
      { type: "song", icon: "🎵", title: "Audacity", author: "Stormzy", detail: "Repeated in chorus as a boast", date: "2026-03-05" },
    ],
    created: "2026-03-09",
    srs: { level: 5, next: "2026-04-02", streak: 8, history: [true, true, true, true, true, false, true, true] },
    notes: "📌 USAGE PATTERNS\nOften used with 'have the audacity to...' — almost always implies surprise or indignation at someone's boldness. The phrase 'sheer audacity' amplifies the emotional charge.\n\n📝 ETYMOLOGY\nFrom Latin 'audacia' (boldness), from 'audax' (bold, daring), from 'audere' (to dare). Related to 'audacious' (adj). The Latin root connects to a broader family of Indo-European words related to eagerness and desire.\n\n⚖️ DUAL MEANING — CONTEXT MATTERS\nPositive sense: courage, willingness to take risks, entrepreneurial spirit. Often used in business/motivational contexts: 'the audacity of hope', 'audacity to dream big'.\nNegative sense: impudence, disrespect, brazenness. More common in everyday speech: 'the audacity of this guy!'\n\n🔗 COLLOCATIONS\n• have the audacity to + verb\n• sheer/pure/absolute audacity\n• audacity of hope (Obama book title, popularized the positive sense)\n• with great audacity\n\n🎬 ENCOUNTERED IN\n— Barack Obama's book title 'The Audacity of Hope' (2006)\n— Heard in The Office S03E07: 'The audacity!' (Michael Scott, negative sense)\n— Article in The Economist about startup founders: 'entrepreneurial audacity'\n\n🤔 PERSONAL ASSOCIATIONS\nReminds me of situations at work where someone challenges a decision everyone else accepted silently. There's a fine line between audacity as courage and audacity as disrespect — it depends entirely on whether the challenger turns out to be right.",
    image: "https://images.unsplash.com/photo-1519834785169-98be25ec3f84?w=400&h=260&fit=crop"
  },
  { id: "3", word: "benevolent", pos: "adj", ipa: "/bəˈnev.əl.ənt/", translation: "доброжелательный", topics: ["Emotions & Feelings"], senses: [{ pos: "Adjective", def: "Well-meaning and kindly", trans: "доброжелательный", examples: [{ en: "A benevolent smile spread across her face.", ru: "Доброжелательная улыбка озарила её лицо." }] }], created: "2026-03-10", srs: { level: 2, next: "2026-03-24", streak: 2, history: [true, true] }, notes: "" },
  { id: "4", word: "catalyst", pos: "n", ipa: "/ˈkæt.əl.ɪst/", translation: "катализатор", topics: ["Academic"], senses: [{ pos: "Noun", def: "A person or thing that precipitates an event or change", trans: "катализатор", examples: [{ en: "The protest was a catalyst for reform.", ru: "Протест стал катализатором реформ." }] }], created: "2026-03-08", srs: { level: 4, next: "2026-03-28", streak: 6, history: [true, false, true, true, true, true] }, notes: "" },
  { id: "5", word: "conundrum", pos: "n", ipa: "/kəˈnʌn.drəm/", translation: "головоломка", topics: ["Advanced Vocabulary"], senses: [{ pos: "Noun", def: "A confusing and difficult problem or question", trans: "головоломка, загадка", examples: [{ en: "The team faced a real conundrum.", ru: "Команда столкнулась с настоящей головоломкой." }] }], created: "2026-03-07", srs: { level: 1, next: "2026-03-23", streak: 1, history: [false, true] }, notes: "" },
  { id: "6", word: "eloquent", pos: "adj", ipa: "/ˈel.ə.kwənt/", translation: "красноречивый", topics: ["Advanced Vocabulary", "Communication"], senses: [{ pos: "Adjective", def: "Fluent or persuasive in speaking or writing", trans: "красноречивый", examples: [{ en: "She gave an eloquent speech at the conference.", ru: "Она произнесла красноречивую речь на конференции." }] }], created: "2026-03-06", srs: { level: 3, next: "2026-03-26", streak: 4, history: [true, true, true, true] }, notes: "" },
  { id: "7", word: "empathy", pos: "n", ipa: "/ˈem.pə.θi/", translation: "эмпатия", topics: ["Emotions & Feelings"], senses: [{ pos: "Noun", def: "The ability to understand and share the feelings of another", trans: "эмпатия, сопереживание", examples: [{ en: "He showed great empathy towards the victims.", ru: "Он проявил большую эмпатию к пострадавшим." }] }], encounters: [{ type: "book", icon: "📚", title: "Nonviolent Communication", author: "M. Rosenberg", detail: "Core concept throughout the book", date: "2026-01-20" }, { type: "tv", icon: "🎬", title: "Inside Out 2", author: "Pixar", detail: "Theme of understanding others' emotions", date: "2026-02-10" }], created: "2026-03-04", srs: { level: 6, next: "2026-04-10", streak: 12, history: [true, true, true, true, true, true] }, notes: "Empathy vs Sympathy: empathy = feeling WITH someone, sympathy = feeling FOR someone." },
  { id: "8", word: "enigma", pos: "n", ipa: "/ɪˈnɪɡ.mə/", translation: "загадка", topics: ["Advanced Vocabulary"], senses: [{ pos: "Noun", def: "A person or thing that is mysterious or difficult to understand", trans: "загадка, тайна", examples: [{ en: "She remained an enigma to all who knew her.", ru: "Она оставалась загадкой для всех, кто её знал." }] }], created: "2026-03-03", srs: { level: 4, next: "2026-03-30", streak: 7, history: [true, true, true, false, true, true, true] }, notes: "" },
  { id: "9", word: "ephemeral", pos: "adj", ipa: "/ɪˈfem.ər.əl/", translation: "мимолётный", topics: ["Advanced Vocabulary", "Literary"], senses: [{ pos: "Adjective", def: "Lasting for a very short time", trans: "мимолётный, эфемерный", examples: [{ en: "Fame in the digital age is often ephemeral.", ru: "Слава в цифровую эпоху часто мимолётна." }] }], created: "2026-03-02", srs: { level: 2, next: "2026-03-24", streak: 3, history: [false, true, true, true] }, notes: "" },
  { id: "10", word: "harmony", pos: "n", ipa: "/ˈhɑː.mə.ni/", translation: "гармония", topics: ["Emotions & Feelings"], senses: [{ pos: "Noun", def: "The state of being in agreement or concord", trans: "гармония, согласие", examples: [{ en: "They lived together in perfect harmony.", ru: "Они жили в полной гармонии." }] }], created: "2026-03-01", srs: { level: 5, next: "2026-04-05", streak: 9, history: [true, true, true, true, true] }, notes: "" },
  { id: "11", word: "lucid", pos: "adj", ipa: "/ˈljuː.sɪd/", translation: "ясный", topics: ["Advanced Vocabulary"], senses: [{ pos: "Adjective", def: "Expressed clearly; easy to understand", trans: "ясный, чёткий", examples: [{ en: "He gave a lucid account of the events.", ru: "Он дал ясное описание событий." }] }], created: "2026-02-28", srs: { level: 3, next: "2026-03-27", streak: 5, history: [true, true, false, true, true] }, notes: "" },
  { id: "12", word: "meticulous", pos: "adj", ipa: "/məˈtɪk.jʊ.ləs/", translation: "скрупулёзный", topics: ["Advanced Vocabulary"], senses: [{ pos: "Adjective", def: "Showing great attention to detail; very careful and precise", trans: "скрупулёзный, дотошный", examples: [{ en: "She kept meticulous records of every transaction.", ru: "Она вела скрупулёзный учёт каждой транзакции." }] }], created: "2026-02-27", srs: { level: 4, next: "2026-03-29", streak: 6, history: [true, true, true, true, true, true] }, notes: "" },
  { id: "13", word: "nostalgia", pos: "n", ipa: "/nɒˈstæl.dʒə/", translation: "ностальгия", topics: ["Emotions & Feelings"], senses: [{ pos: "Noun", def: "A sentimental longing for the past", trans: "ностальгия, тоска по прошлому", examples: [{ en: "The old photographs filled her with nostalgia.", ru: "Старые фотографии наполнили её ностальгией." }] }], created: "2026-02-26", srs: { level: 5, next: "2026-04-01", streak: 10, history: [true, true, true, true, true] }, notes: "" },
  { id: "14", word: "quintessential", pos: "adj", ipa: "/ˌkwɪn.tɪˈsen.ʃəl/", translation: "типичнейший", topics: ["Advanced Vocabulary", "Literary"], senses: [{ pos: "Adjective", def: "Representing the most perfect or typical example of a quality", trans: "типичнейший, квинтэссенциальный", examples: [{ en: "He was the quintessential English gentleman.", ru: "Он был типичнейшим английским джентльменом." }] }], created: "2026-02-25", srs: { level: 1, next: "2026-03-23", streak: 0, history: [false, false] }, notes: "" },
  { id: "15", word: "resilient", pos: "adj", ipa: "/rɪˈzɪl.i.ənt/", translation: "стойкий", topics: ["Emotions & Feelings", "Academic"], senses: [{ pos: "Adjective", def: "Able to recover quickly from difficult conditions", trans: "стойкий, жизнестойкий", examples: [{ en: "Children are often more resilient than adults.", ru: "Дети часто более стойкие, чем взрослые." }] }], created: "2026-02-24", srs: { level: 3, next: "2026-03-25", streak: 4, history: [true, true, true, true] }, notes: "" },
];

const NAV = [
  { label: "Dashboard", icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg> },
  { label: "Dictionary", icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>, active: true },
  { label: "Study", icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg> },
  { label: "Topics", icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg> },
  { label: "Inbox", icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></svg> },
];

/* ── PALETTE ──
   Surface hierarchy (lightness):
   pageBg 87% → cardBg 100% (+13% jump, strong elevation)
   cardBg 100% → inset 95% (visible recessed areas)
   inset 95% → nested white 100% (example blocks pop)
   footer 92% (clearly distinct band)
*/
const P = {
  sidebarBg: "#1B1D23", sidebarText: "#9499A8", sidebarTextActive: "#FFFFFF",
  sidebarActiveBg: "rgba(99,130,202,0.15)", sidebarAccent: "#6382CA", sidebarBorder: "#2A2D36",
  // Surfaces — clear jumps between layers
  pageBg: "#D8DCE5",        // cool blue-gray, clearly darker
  cardBg: "#FFFFFF",         // pure white — big contrast with page
  surfaceInset: "#EDF0F5",   // cool-tinted, visible on white
  surfaceFooter: "#E4E7ED",  // distinct footer band
  // Text — 4 clear levels
  textPrimary: "#14161E",    // near-black for max readability
  textSecondary: "#4A5068",  // clearly readable secondary
  textTertiary: "#788094",   // de-emphasized but legible
  textMuted: "#A4AAB8",      // decorative, indices
  // Accent
  accent: "#5872A8",         // muted slate-blue
  accentLight: "#E8ECF4",    // subtle blue tint
  accentBorder: "#C2CCDF",   // soft
  // Semantic — desaturated, calm tones
  green: "#4D8A6F", greenBg: "#E4EFE9", greenBorder: "#B4D0C0",
  orange: "#9E7E4A", orangeBg: "#F2ECE0", orangeBorder: "#D8CCAF",
  red: "#B06068", redBg: "#F4E8EA", redBorder: "#DDB8BD",
  purple: "#7E6DA0", purpleBg: "#EDEAF4", purpleBorder: "#C8C0DC",
  teal: "#4E8585", tealBg: "#E4EEEE", tealBorder: "#B4D0D0",
  // Borders — stronger
  border: "#C6CBD6",         // clearly visible
  borderLight: "#D8DCE5",    // lighter but still visible
};

const TOPIC_COLORS = {
  "Advanced Vocabulary": { bg: P.purpleBg, color: P.purple, border: P.purpleBorder },
  "Emotions & Feelings": { bg: P.orangeBg, color: P.orange, border: P.orangeBorder },
  "Academic": { bg: P.tealBg, color: P.teal, border: P.tealBorder },
  "Communication": { bg: P.greenBg, color: P.green, border: P.greenBorder },
  "Literary": { bg: "#F0E6EB", color: "#907080", border: "#D4C0CA" },
};

const SRS_LEVELS = [
  { label: "New", color: P.textTertiary, bg: P.surfaceInset },
  { label: "Learning", color: P.orange, bg: P.orangeBg },
  { label: "Familiar", color: "#8A7A4A", bg: "#F0EBE0" },
  { label: "Known", color: P.teal, bg: P.tealBg },
  { label: "Strong", color: P.accent, bg: P.accentLight },
  { label: "Mastered", color: P.green, bg: P.greenBg },
  { label: "Locked", color: P.purple, bg: P.purpleBg },
];

/* ─────────── HELPERS ─────────── */
function groupByLetter(words) {
  const g = {};
  words.forEach(w => { const l = w.word[0].toUpperCase(); (g[l] = g[l] || []).push(w); });
  return Object.entries(g).sort(([a], [b]) => a.localeCompare(b));
}

function SrsBar({ history }) {
  return (
    <div style={{ display: "flex", gap: 2, alignItems: "end" }}>
      {history.slice(-10).map((ok, i) => (
        <div key={i} style={{
          width: 5, height: ok ? 16 : 10, borderRadius: 2,
          background: ok ? P.green : P.red,
          opacity: 0.35 + (i / Math.max(history.length - 1, 1)) * 0.65,
        }} />
      ))}
    </div>
  );
}

function SrsLevel({ level }) {
  const s = SRS_LEVELS[level];
  return (
    <span style={{
      display: "inline-flex", alignItems: "center", gap: 5,
      padding: "2px 9px", borderRadius: 5, fontSize: 11, fontWeight: 600,
      background: s.bg, color: s.color,
    }}>
      <span style={{ width: 6, height: 6, borderRadius: "50%", background: s.color }} />
      {s.label}
    </span>
  );
}

function PosTag({ pos }) {
  return (
    <span style={{
      fontSize: 10, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em",
      padding: "2px 6px", borderRadius: 4,
      border: `1px solid ${P.border}`,
      color: P.textTertiary,
    }}>{pos}</span>
  );
}


/* ─────────── EDITABLE INPUT HELPERS ─────────── */
function EditInput({ value, onChange, style: s = {}, mono, placeholder }) {
  return (
    <input type="text" value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder}
      style={{
        border: "none", outline: "none", background: P.cardBg, borderRadius: 5,
        padding: "4px 8px", fontSize: 13, color: P.textPrimary, width: "100%",
        fontFamily: mono ? "'JetBrains Mono', monospace" : "inherit",
        boxShadow: `inset 0 0 0 1px ${P.accentBorder}`,
        transition: "box-shadow 0.12s", ...s,
      }}
      onFocus={e => e.currentTarget.style.boxShadow = `inset 0 0 0 1.5px ${P.accent}`}
      onBlur={e => e.currentTarget.style.boxShadow = `inset 0 0 0 1px ${P.accentBorder}`}
    />
  );
}
function EditTextarea({ value, onChange, style: s = {}, placeholder, rows = 2 }) {
  return (
    <textarea value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder} rows={rows}
      style={{
        border: "none", outline: "none", background: P.cardBg, borderRadius: 5,
        padding: "6px 8px", fontSize: 13, color: P.textPrimary, width: "100%",
        fontFamily: "inherit", resize: "vertical", lineHeight: 1.5,
        boxShadow: `inset 0 0 0 1px ${P.accentBorder}`,
        transition: "box-shadow 0.12s", ...s,
      }}
      onFocus={e => e.currentTarget.style.boxShadow = `inset 0 0 0 1.5px ${P.accent}`}
      onBlur={e => e.currentTarget.style.boxShadow = `inset 0 0 0 1px ${P.accentBorder}`}
    />
  );
}


/* ─────────── WORD CARD — BENTO ─────────── */
function WordCard({ word, onClose }) {
  const [editing, setEditing] = useState(false);
  const [expanded, setExpanded] = useState(null);
  const makeSenses = () => word.senses.map(s => ({ pos: s.pos, def: s.def, trans: s.trans, examples: s.examples.map(ex => ({ en: ex.en, ru: ex.ru })) }));
  const [form, setForm] = useState({ word: word.word, ipa: word.ipa, pos: word.pos, topics: [...word.topics], senses: makeSenses(), notes: word.notes || "" });

  const successRate = word.srs.history.length > 0 ? Math.round(word.srs.history.filter(Boolean).length / word.srs.history.length * 100) : 0;
  const posLabel = word.pos === "n" ? "noun" : word.pos === "adj" ? "adjective" : word.pos;
  const encounters = word.encounters || [];

  const updateSense = (idx, key, val) => { const s = [...form.senses]; s[idx] = { ...s[idx], [key]: val }; setForm({ ...form, senses: s }); };
  const updateExample = (si, ei, key, val) => { const s = [...form.senses]; const exs = [...s[si].examples]; exs[ei] = { ...exs[ei], [key]: val }; s[si] = { ...s[si], examples: exs }; setForm({ ...form, senses: s }); };
  const addSense = () => setForm({ ...form, senses: [...form.senses, { pos: "Noun", def: "", trans: "", examples: [{ en: "", ru: "" }] }] });
  const removeSense = (idx) => { if (form.senses.length > 1) { const s = [...form.senses]; s.splice(idx, 1); setForm({ ...form, senses: s }); } };
  const addExample = (si) => { const s = [...form.senses]; s[si] = { ...s[si], examples: [...s[si].examples, { en: "", ru: "" }] }; setForm({ ...form, senses: s }); };
  const cancelEdit = () => { setEditing(false); setForm({ word: word.word, ipa: word.ipa, pos: word.pos, topics: [...word.topics], senses: makeSenses(), notes: word.notes || "" }); };

  const cell = { background: P.surfaceInset, borderRadius: 10, padding: "14px 16px", overflow: "hidden", minWidth: 0 };
  const sLabel = { fontSize: 10, fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.07em", color: P.textMuted, marginBottom: 8, display: "flex", alignItems: "center", gap: 5 };
  const ib = { background: "none", border: "none", cursor: "pointer", width: 28, height: 28, borderRadius: 6, display: "flex", alignItems: "center", justifyContent: "center", color: P.textMuted, transition: "all 0.1s" };

  const moreBtn = (key, count) => (
    <button onClick={() => setExpanded(expanded === key ? null : key)} style={{ background: "none", border: "none", padding: "6px 0 0", fontSize: 11, fontWeight: 600, color: P.textMuted, cursor: "pointer", transition: "color 0.12s" }}
      onMouseEnter={e => e.currentTarget.style.color = P.accent} onMouseLeave={e => e.currentTarget.style.color = P.textMuted}>
      {count ? `Show all ${count} →` : "Show more →"}
    </button>
  );

  const EncRow = ({ enc, compact }) => (
    <div style={{ display: "flex", alignItems: "flex-start", gap: compact ? 8 : 12, padding: compact ? "7px 10px" : "12px 14px", background: P.cardBg, borderRadius: compact ? 7 : 8, border: compact ? "none" : `1px solid ${P.borderLight}`, transition: "all 0.1s", cursor: "pointer" }}
      onMouseEnter={e => e.currentTarget.style.boxShadow = "0 1px 4px rgba(20,22,30,0.06)"} onMouseLeave={e => e.currentTarget.style.boxShadow = "none"}>
      <span style={{ fontSize: compact ? 16 : 22, lineHeight: 1, flexShrink: 0, marginTop: 1 }}>{enc.icon}</span>
      <div style={{ minWidth: 0, flex: 1 }}>
        <div style={{ display: "flex", alignItems: "baseline", gap: 5 }}>
          <span style={{ fontSize: compact ? 12 : 14, fontWeight: 600, color: P.textPrimary, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{enc.title}</span>
          <span style={{ fontSize: compact ? 10 : 11, color: P.textMuted, flexShrink: 0 }}>{enc.author}</span>
        </div>
        <p style={{ fontSize: compact ? 11 : 13, color: P.textTertiary, margin: "2px 0 0", lineHeight: 1.4, overflow: compact ? "hidden" : "visible", textOverflow: compact ? "ellipsis" : "clip", whiteSpace: compact ? "nowrap" : "normal" }}>{enc.detail}</p>
      </div>
      <span style={{ fontSize: compact ? 9 : 11, color: P.textMuted, flexShrink: 0, marginTop: 2 }}>{compact ? enc.date.slice(5) : enc.date}</span>
    </div>
  );

  return (
    <div style={{
      background: P.cardBg, borderRadius: 14, border: `1px solid ${editing ? P.accentBorder : P.border}`,
      boxShadow: editing ? `0 6px 28px rgba(20,22,30,0.10), 0 0 0 2px ${P.accent}20` : "0 6px 28px rgba(20,22,30,0.10), 0 2px 6px rgba(20,22,30,0.06)",
      margin: "6px 0 14px 0", overflow: "hidden", animation: "cardIn 0.2s ease-out", padding: 10,
    }}>

      {/* ═══ HERO ═══ */}
      <div style={{ ...cell, padding: "18px 20px 16px", marginBottom: 8 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            {editing ? (
              <>
                <div style={{ display: "flex", gap: 8, marginBottom: 8 }}>
                  <EditInput value={form.word} onChange={v => setForm({ ...form, word: v })} style={{ fontSize: 22, fontWeight: 700, fontFamily: "'Source Serif 4', Georgia, serif", flex: 2 }} />
                  <EditInput value={form.ipa} onChange={v => setForm({ ...form, ipa: v })} mono style={{ fontSize: 12, flex: 1 }} placeholder="/IPA/" />
                </div>
                <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                  <select value={form.pos} onChange={e => setForm({ ...form, pos: e.target.value })} style={{ border: "none", background: P.cardBg, borderRadius: 5, padding: "3px 8px", fontSize: 11, color: P.textSecondary, outline: "none", cursor: "pointer", boxShadow: `inset 0 0 0 1px ${P.accentBorder}` }}>
                    <option value="n">noun</option><option value="adj">adjective</option><option value="v">verb</option><option value="adv">adverb</option>
                  </select>
                  <EditInput value={form.topics.join(", ")} onChange={v => setForm({ ...form, topics: v.split(",").map(t => t.trim()).filter(Boolean) })} placeholder="Topics, comma separated" style={{ fontSize: 11, flex: 1 }} />
                </div>
              </>
            ) : (
              <>
                <div style={{ display: "flex", alignItems: "baseline", gap: 10 }}>
                  <h2 style={{ fontSize: 28, fontWeight: 700, color: P.textPrimary, margin: 0, fontFamily: "'Source Serif 4', Georgia, serif", letterSpacing: "-0.03em" }}>{word.word}</h2>
                  <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12.5, color: P.textTertiary }}>{word.ipa}</span>
                  <span style={{ fontSize: 11, color: P.textMuted }}>{posLabel}</span>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 10, flexWrap: "wrap" }}>
                  <SrsLevel level={word.srs.level} />
                  {word.topics.map(t => { const c = TOPIC_COLORS[t] || { bg: P.surfaceInset, color: P.textTertiary, border: P.border }; return <span key={t} style={{ padding: "2px 7px", borderRadius: 4, fontSize: 10, fontWeight: 500, background: c.bg, color: c.color, border: `1px solid ${c.border}` }}>{t}</span>; })}
                  <span style={{ fontSize: 10, color: P.textMuted, marginLeft: 4 }}>added {word.created}</span>
                </div>
              </>
            )}
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 2, flexShrink: 0, marginLeft: 10 }}>
            {editing ? (
              <>
                <button title="Save" onClick={() => setEditing(false)} style={{ ...ib, background: P.accent, color: "#fff" }} onMouseEnter={e => e.currentTarget.style.opacity = "0.85"} onMouseLeave={e => e.currentTarget.style.opacity = "1"}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg></button>
                <button title="Cancel" onClick={cancelEdit} style={ib} onMouseEnter={e => { e.currentTarget.style.background = P.cardBg; e.currentTarget.style.color = P.textSecondary; }} onMouseLeave={e => { e.currentTarget.style.background = "none"; e.currentTarget.style.color = P.textMuted; }}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M3 12h18M3 12l6-6M3 12l6 6"/></svg></button>
              </>
            ) : (
              <>
                <button title="Edit" onClick={() => setEditing(true)} style={ib} onMouseEnter={e => { e.currentTarget.style.background = P.cardBg; e.currentTarget.style.color = P.textSecondary; }} onMouseLeave={e => { e.currentTarget.style.background = "none"; e.currentTarget.style.color = P.textMuted; }}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M17 3a2.83 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg></button>
                <button title="Delete" style={ib} onMouseEnter={e => { e.currentTarget.style.background = P.cardBg; e.currentTarget.style.color = P.red; }} onMouseLeave={e => { e.currentTarget.style.background = "none"; e.currentTarget.style.color = P.textMuted; }}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg></button>
                <div style={{ width: 1, height: 16, background: P.border, margin: "0 4px" }} />
                <button onClick={onClose} title="Close" style={ib} onMouseEnter={e => { e.currentTarget.style.background = P.cardBg; e.currentTarget.style.color = P.textPrimary; }} onMouseLeave={e => { e.currentTarget.style.background = "none"; e.currentTarget.style.color = P.textMuted; }}>
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
              </>
            )}
          </div>
        </div>
      </div>


      {/* ═══ 3-COL BENTO: defs(2col) + widgets(1col), bottom row fills gap ═══ */}
      <div style={{
        display: "grid",
        gridTemplateColumns: "1fr 1fr 220px",
        gridTemplateRows: "auto auto",
        gridTemplateAreas: `"defs defs widgets" "enc notes widgets"`,
        gap: 8,
      }}>

        {/* DEFINITIONS — spans 2 columns */}
        <div style={{ ...cell, gridArea: "defs", background: P.cardBg, border: `1px solid ${P.border}`, padding: "16px 18px" }}>
          <div style={{ ...sLabel, justifyContent: "space-between" }}>
            <span style={{ display: "flex", alignItems: "center", gap: 5 }}><span>📖</span> Definitions</span>
            {editing && <button onClick={addSense} style={{ background: "none", border: `1px solid ${P.border}`, borderRadius: 5, padding: "2px 8px", fontSize: 10, fontWeight: 600, color: P.accent, cursor: "pointer" }}
              onMouseEnter={e => { e.currentTarget.style.borderColor = P.accent; e.currentTarget.style.background = P.accentLight; }} onMouseLeave={e => { e.currentTarget.style.borderColor = P.border; e.currentTarget.style.background = "none"; }}>+ Add sense</button>}
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            {(editing ? form.senses : word.senses).map((s, i) => (
              <div key={i} style={{ paddingLeft: 12, borderLeft: `3px solid ${i === 0 ? P.accent : P.border}` }}>
                <div style={{ display: "flex", alignItems: "center", gap: 5, marginBottom: 4 }}>
                  <span style={{ fontSize: 11, fontWeight: 700, color: P.accent }}>{i + 1}</span>
                  {editing ? <select value={s.pos} onChange={e => updateSense(i, "pos", e.target.value)} style={{ border: "none", background: P.cardBg, borderRadius: 4, padding: "1px 4px", fontSize: 10, color: P.textSecondary, outline: "none", boxShadow: `inset 0 0 0 1px ${P.accentBorder}` }}><option>Noun</option><option>Adjective</option><option>Verb</option><option>Adverb</option></select>
                    : <span style={{ fontSize: 10, fontWeight: 600, textTransform: "uppercase", color: P.textMuted }}>{s.pos}</span>}
                  {editing && form.senses.length > 1 && <button onClick={() => removeSense(i)} style={{ marginLeft: "auto", background: "none", border: "none", cursor: "pointer", color: P.textMuted, fontSize: 14, padding: "0 4px" }} onMouseEnter={e => e.currentTarget.style.color = P.red} onMouseLeave={e => e.currentTarget.style.color = P.textMuted}>×</button>}
                </div>
                {editing ? (<><EditInput value={s.def} onChange={v => updateSense(i, "def", v)} placeholder="Definition" style={{ marginBottom: 4, fontSize: 13 }} /><EditInput value={s.trans} onChange={v => updateSense(i, "trans", v)} placeholder="Перевод" style={{ marginBottom: 8, fontSize: 12 }} /></>)
                  : (<><p style={{ fontSize: 13.5, lineHeight: 1.55, color: P.textPrimary, margin: "0 0 2px" }}>{s.def}</p><p style={{ fontSize: 12.5, color: P.accent, margin: "0 0 8px", fontWeight: 500 }}>{s.trans}</p></>)}
                {s.examples.map((ex, j) => (
                  <div key={j} style={{ background: P.surfaceInset, borderRadius: 6, padding: editing ? "6px 8px" : "8px 10px", marginTop: j > 0 ? 4 : 0 }}>
                    {editing ? (<><EditInput value={ex.en} onChange={v => updateExample(i, j, "en", v)} placeholder="Example" style={{ fontSize: 12, marginBottom: 3, background: P.surfaceInset }} /><EditInput value={ex.ru} onChange={v => updateExample(i, j, "ru", v)} placeholder="Перевод" style={{ fontSize: 11, background: P.surfaceInset }} /></>)
                      : (<><p style={{ fontSize: 12.5, color: P.textPrimary, margin: 0, lineHeight: 1.45, fontStyle: "italic" }}>"{ex.en}"</p><p style={{ fontSize: 11.5, color: P.textTertiary, margin: "2px 0 0" }}>{ex.ru}</p></>)}
                  </div>
                ))}
                {editing && <button onClick={() => addExample(i)} style={{ background: "none", border: "none", fontSize: 10, color: P.textMuted, cursor: "pointer", padding: "4px 0" }} onMouseEnter={e => e.currentTarget.style.color = P.accent} onMouseLeave={e => e.currentTarget.style.color = P.textMuted}>+ example</button>}
              </div>
            ))}
          </div>
        </div>

        {/* RIGHT COLUMN — widgets stack, spans both rows */}
        <div style={{ gridArea: "widgets", display: "flex", flexDirection: "column", gap: 8 }}>
          {/* MEDIA */}
          <div style={{ ...cell, padding: 0 }}>
            <div style={{ height: 80, position: "relative" }}>
              {word.image ? (<><img src={word.image} alt={word.word} style={{ width: "100%", height: "100%", objectFit: "cover", display: "block", borderRadius: "10px 10px 0 0" }} />{editing && <button style={{ position: "absolute", top: 4, right: 4, background: "rgba(0,0,0,0.5)", border: "none", color: "#fff", width: 20, height: 20, borderRadius: 4, cursor: "pointer", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 12 }}>×</button>}</>)
                : (<div style={{ width: "100%", height: "100%", background: `linear-gradient(135deg, ${P.surfaceInset}, ${P.surfaceFooter})`, display: "flex", alignItems: "center", justifyContent: "center", borderRadius: "10px 10px 0 0", cursor: "pointer", color: P.textMuted }}><span style={{ fontSize: 20 }}>🖼️</span></div>)}
            </div>
            <div style={{ display: "flex", gap: 1, background: P.cardBg, borderTop: `1px solid ${P.borderLight}` }}>
              {["🇬🇧 UK", "🇺🇸 US"].map((v, i) => (
                <button key={v} style={{ flex: 1, padding: "6px 0", border: "none", background: P.surfaceInset, cursor: "pointer", fontSize: 10, fontWeight: 600, color: P.textTertiary, transition: "all 0.1s", borderRadius: i === 0 ? "0 0 0 10px" : "0 0 10px 0" }}
                  onMouseEnter={e => { e.currentTarget.style.background = P.accentLight; e.currentTarget.style.color = P.accent; }} onMouseLeave={e => { e.currentTarget.style.background = P.surfaceInset; e.currentTarget.style.color = P.textTertiary; }}>{v}</button>
              ))}
            </div>
          </div>
          {/* PROGRESS */}
          <div style={cell}>
            <div style={sLabel}><span>📊</span> Progress</div>
            <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6, padding: "4px 6px", background: P.cardBg, borderRadius: 6 }}>
              <span style={{ fontSize: 18 }}>🔥</span>
              <div><div style={{ fontSize: 16, fontWeight: 800, color: P.textPrimary, lineHeight: 1 }}>{word.srs.streak}</div><div style={{ fontSize: 9, color: P.textMuted }}>streak</div></div>
              <div style={{ marginLeft: "auto", textAlign: "right" }}><div style={{ fontSize: 13, fontWeight: 700, color: P.textPrimary, lineHeight: 1 }}>{successRate}%</div><div style={{ fontSize: 9, color: P.textMuted }}>accuracy</div></div>
            </div>
            <div style={{ display: "flex", gap: 2, alignItems: "end" }}>
              {word.srs.history.slice(-12).map((ok, i, arr) => (<div key={i} style={{ flex: 1, height: ok ? 14 : 7, borderRadius: 2, background: ok ? P.green : P.red, opacity: 0.35 + (i / Math.max(arr.length - 1, 1)) * 0.65 }} />))}
            </div>
          </div>
        </div>

        {/* BOTTOM LEFT: Encounters */}
        <div style={{ ...cell, gridArea: "enc" }}>
          <div style={sLabel}><span>📍</span> Encounters {encounters.length > 0 && <span style={{ fontWeight: 500, letterSpacing: 0 }}>({encounters.length})</span>}</div>
          {encounters.length > 0 ? (<>
            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>{encounters.slice(0, 2).map((enc, i) => <EncRow key={i} enc={enc} compact />)}</div>
            {encounters.length > 2 && moreBtn("encounters", encounters.length)}
          </>) : (
            <div style={{ padding: 10, borderRadius: 6, border: `1px dashed ${P.border}`, textAlign: "center", color: P.textMuted, fontSize: 10, cursor: "pointer" }}><span style={{ fontSize: 14, display: "block", marginBottom: 2 }}>📍</span>Log encounters</div>
          )}
        </div>

        {/* BOTTOM CENTER: Notes */}
        <div style={{ ...cell, gridArea: "notes" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 5, marginBottom: 6 }}><span style={{ fontSize: 12 }}>✏️</span><span style={{ fontSize: 10, fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.07em", color: P.textMuted }}>Notes</span></div>
          {editing ? <EditTextarea value={form.notes} onChange={v => setForm({ ...form, notes: v })} placeholder="Your notes..." rows={3} style={{ fontSize: 12, background: P.surfaceInset }} />
            : word.notes ? (<><p style={{ fontSize: 12, lineHeight: 1.5, color: P.textSecondary, margin: 0, whiteSpace: "pre-wrap", display: "-webkit-box", WebkitLineClamp: 3, WebkitBoxOrient: "vertical", overflow: "hidden" }}>{word.notes}</p>{moreBtn("notes")}</>)
              : <p style={{ fontSize: 12, color: P.textMuted, margin: 0, cursor: "pointer", fontStyle: "italic" }}>Add notes...</p>}
        </div>
      </div>

      {/* ═══ EXPANDED PANEL ═══ */}
      {expanded && (
        <div style={{ marginTop: 8, background: P.surfaceInset, borderRadius: 10, padding: "16px 20px", animation: "cardIn 0.15s ease-out" }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 14 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <span style={{ fontSize: 14 }}>{expanded === "notes" ? "✏️" : "📍"}</span>
              <span style={{ fontSize: 11, fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.06em", color: P.textTertiary }}>{expanded === "notes" ? "Notes" : "Encounters"}</span>
              {expanded === "encounters" && <span style={{ fontSize: 11, color: P.textMuted }}>({encounters.length})</span>}
            </div>
            <button onClick={() => setExpanded(null)} style={{ background: "none", border: `1px solid ${P.border}`, borderRadius: 5, padding: "3px 10px", fontSize: 10, fontWeight: 600, color: P.textTertiary, cursor: "pointer", transition: "all 0.12s" }}
              onMouseEnter={e => { e.currentTarget.style.color = P.textSecondary; e.currentTarget.style.borderColor = P.textTertiary; }}
              onMouseLeave={e => { e.currentTarget.style.color = P.textTertiary; e.currentTarget.style.borderColor = P.border; }}>← Collapse</button>
          </div>
          {expanded === "notes" && (editing
            ? <EditTextarea value={form.notes} onChange={v => setForm({ ...form, notes: v })} placeholder="Your notes..." rows={10} style={{ fontSize: 13, background: P.surfaceInset }} />
            : <p style={{ fontSize: 13, lineHeight: 1.7, color: P.textPrimary, margin: 0, whiteSpace: "pre-wrap" }}>{word.notes || ""}</p>
          )}
          {expanded === "encounters" && <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>{encounters.map((enc, i) => <EncRow key={i} enc={enc} />)}</div>}
        </div>
      )}
    </div>
  );
}

function WordRow({ word, index, isOpen, onToggle }) {
  if (isOpen) {
    return <WordCard word={word} onClose={onToggle} />;
  }
  return (
    <div onClick={onToggle} style={{
      display: "grid", gridTemplateColumns: "28px 1fr auto", alignItems: "center",
      padding: "9px 12px", borderRadius: 8, cursor: "pointer", transition: "all 0.1s",
      background: "transparent", border: "1px solid transparent",
    }}
      onMouseEnter={e => { e.currentTarget.style.background = P.cardBg; e.currentTarget.style.boxShadow = "0 1px 4px rgba(20,22,30,0.06)"; e.currentTarget.style.border = `1px solid ${P.border}`; }}
      onMouseLeave={e => { e.currentTarget.style.background = "transparent"; e.currentTarget.style.boxShadow = "none"; e.currentTarget.style.border = "1px solid transparent"; }}>
      <span style={{ fontSize: 11, color: P.textMuted, fontFamily: "'JetBrains Mono', monospace", fontWeight: 500 }}>{index}</span>
      <div style={{ display: "flex", alignItems: "baseline", gap: 10, minWidth: 0 }}>
        <span style={{ fontSize: 15, fontWeight: 600, color: P.textPrimary, flexShrink: 0 }}>{word.word}</span>
        <PosTag pos={word.pos} />
        <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 11.5, color: P.textMuted, flexShrink: 0 }}>{word.ipa}</span>
        <span style={{ color: P.borderLight }}>—</span>
        <span style={{ fontSize: 13.5, color: P.textSecondary, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{word.translation}</span>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <SrsBar history={word.srs.history} />
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke={P.textMuted} strokeWidth="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </div>
    </div>
  );
}

/* ─────────── MAIN PAGE ─────────── */
export default function DictionaryPage() {
  const [openId, setOpenId] = useState("2");
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState("a-z");

  const filtered = WORDS.filter(w =>
    w.word.toLowerCase().includes(search.toLowerCase()) ||
    w.translation.toLowerCase().includes(search.toLowerCase())
  );
  const sorted = [...filtered].sort((a, b) => {
    if (sort === "a-z") return a.word.localeCompare(b.word);
    if (sort === "new") return new Date(b.created) - new Date(a.created);
    return 0;
  });
  const grouped = groupByLetter(sorted);
  const nounCount = filtered.filter(w => w.pos === "n").length;
  const adjCount = filtered.filter(w => w.pos === "adj").length;

  return (
    <div style={{ display: "flex", height: "100vh", fontFamily: "'DM Sans', system-ui, sans-serif", background: P.pageBg, color: P.textPrimary }}>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@300;400;500;600;700&family=Source+Serif+4:opsz,wght@8..60,400;8..60,600;8..60,700&family=JetBrains+Mono:wght@400;500&display=swap');
        @keyframes cardIn { from { opacity: 0; transform: translateY(-6px); } to { opacity: 1; transform: translateY(0); } }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        ::-webkit-scrollbar { width: 6px; }
        ::-webkit-scrollbar-track { background: transparent; }
        ::-webkit-scrollbar-thumb { background: ${P.textMuted}40; border-radius: 3px; }
        ::selection { background: ${P.accent}25; }
      `}</style>

      {/* Sidebar */}
      <aside style={{ width: 224, background: P.sidebarBg, display: "flex", flexDirection: "column", padding: "16px 10px", flexShrink: 0, borderRight: `1px solid ${P.sidebarBorder}` }}>
        <div style={{ fontSize: 17, fontWeight: 700, color: P.sidebarTextActive, padding: "8px 12px", marginBottom: 24, fontFamily: "'Source Serif 4', Georgia, serif", display: "flex", alignItems: "center", gap: 8 }}>
          <span style={{ width: 24, height: 24, borderRadius: 6, background: `linear-gradient(135deg, ${P.accent}, ${P.purple})`, display: "flex", alignItems: "center", justifyContent: "center", fontSize: 12, color: "#fff", fontWeight: 800 }}>M</span>
          MyEnglish
        </div>
        <nav style={{ display: "flex", flexDirection: "column", gap: 2, flex: 1 }}>
          {NAV.map(n => (
            <div key={n.label} style={{
              display: "flex", alignItems: "center", gap: 10, padding: "8px 12px", borderRadius: 7, cursor: "pointer",
              fontSize: 14, fontWeight: n.active ? 550 : 400,
              color: n.active ? P.sidebarTextActive : P.sidebarText,
              background: n.active ? P.sidebarActiveBg : "transparent",
            }}
              onMouseEnter={e => { if (!n.active) { e.currentTarget.style.color = P.sidebarTextActive; e.currentTarget.style.background = "rgba(255,255,255,0.05)"; } }}
              onMouseLeave={e => { if (!n.active) { e.currentTarget.style.color = P.sidebarText; e.currentTarget.style.background = "transparent"; } }}>
              {n.icon}{n.label}
              {n.label === "Inbox" && <span style={{ marginLeft: "auto", fontSize: 10, fontWeight: 700, background: P.red, color: "#fff", padding: "1px 6px", borderRadius: 10 }}>3</span>}
            </div>
          ))}
        </nav>
        <div style={{ borderTop: `1px solid ${P.sidebarBorder}`, paddingTop: 10 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "8px 12px", fontSize: 13, color: P.sidebarText, cursor: "pointer", borderRadius: 6 }}>
            <div style={{ width: 26, height: 26, borderRadius: 7, background: `linear-gradient(135deg, ${P.accent}, ${P.purple})`, display: "flex", alignItems: "center", justifyContent: "center", fontSize: 11, color: "#fff", fontWeight: 700 }}>NK</div>
            Settings
          </div>
        </div>
      </aside>

      {/* Main */}
      <main style={{ flex: 1, overflow: "auto", padding: "32px 44px" }}>
        <div style={{ maxWidth: 860, margin: "0 auto" }}>
          <div style={{ marginBottom: 24 }}>
            <h1 style={{ fontSize: 26, fontWeight: 700, color: P.textPrimary, margin: "0 0 8px", fontFamily: "'Source Serif 4', Georgia, serif", letterSpacing: "-0.02em" }}>Dictionary</h1>
            <div style={{ display: "flex", alignItems: "center", gap: 14, fontSize: 13, color: P.textTertiary }}>
              <span><strong style={{ color: P.textSecondary, fontWeight: 600 }}>{filtered.length}</strong> words</span>
              <span style={{ width: 3, height: 3, borderRadius: "50%", background: P.textMuted }} />
              <span><strong style={{ color: P.textSecondary, fontWeight: 600 }}>{nounCount}</strong> nouns</span>
              <span><strong style={{ color: P.textSecondary, fontWeight: 600 }}>{adjCount}</strong> adj</span>
              <span style={{ width: 3, height: 3, borderRadius: "50%", background: P.textMuted }} />
              <span>{[...new Set(WORDS.flatMap(w => w.topics))].length} topics</span>
            </div>
          </div>

          <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 24, padding: "10px 14px", background: P.cardBg, borderRadius: 10, border: `1px solid ${P.border}`, boxShadow: "0 2px 8px rgba(20,22,30,0.06)" }}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke={P.textTertiary} strokeWidth="2"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
            <input type="text" placeholder="Search words..." value={search} onChange={e => setSearch(e.target.value)}
              style={{ flex: 1, border: "none", outline: "none", fontSize: 14, background: "transparent", color: P.textPrimary, fontFamily: "inherit" }} />
            <div style={{ height: 20, width: 1, background: P.border }} />
            <div style={{ display: "flex", gap: 2, background: P.surfaceInset, borderRadius: 6, padding: 2 }}>
              {[{ k: "a-z", l: "A–Z" }, { k: "new", l: "Newest" }].map(s => (
                <button key={s.k} onClick={() => setSort(s.k)} style={{
                  padding: "4px 14px", borderRadius: 5, fontSize: 12, fontWeight: 600, border: "none", cursor: "pointer",
                  background: sort === s.k ? P.cardBg : "transparent", color: sort === s.k ? P.textPrimary : P.textTertiary,
                  boxShadow: sort === s.k ? "0 1px 3px rgba(0,0,0,0.06)" : "none",
                }}>{s.l}</button>
              ))}
            </div>
          </div>

          <div style={{ display: "flex", flexDirection: "column" }}>
            {grouped.map(([letter, words]) => (
              <div key={letter} style={{ marginBottom: 4 }}>
                <div style={{ fontSize: 12, fontWeight: 700, color: P.accent, padding: "10px 12px 4px", letterSpacing: "0.06em", position: "sticky", top: 0, background: P.pageBg, zIndex: 5, display: "flex", alignItems: "center", gap: 10 }}>
                  {letter}<div style={{ flex: 1, height: 1, background: P.border }} />
                </div>
                {words.map(w => (
                  <WordRow key={w.id} word={w} index={WORDS.indexOf(w) + 1}
                    isOpen={openId === w.id} onToggle={() => setOpenId(openId === w.id ? null : w.id)} />
                ))}
              </div>
            ))}
          </div>
        </div>
      </main>
    </div>
  );
}
