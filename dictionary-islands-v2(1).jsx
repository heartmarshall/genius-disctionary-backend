import { useState } from "react";

const TOPICS = {
  "Advanced Vocabulary": { color: "#7b9ec7", bg: "#7b9ec714" },
  "Emotions & Feelings": { color: "#c48b7a", bg: "#c48b7a14" },
  "Academic English": { color: "#8aaa7e", bg: "#8aaa7e14" },
  "Business & Work": { color: "#c4ab6e", bg: "#c4ab6e14" },
  "Daily Life": { color: "#a68eb8", bg: "#a68eb814" },
};

const SRS = {
  new:      { label: "new",      color: "#706a60" },
  learning: { label: "learning", color: "#c4ab6e" },
  review:   { label: "review",   color: "#7b9ec7" },
  mastered: { label: "mastered", color: "#8aaa7e" },
};

const words = [
  { id:1, word:"ambiguous", ipa:"/æmˈbɪɡ.ju.əs/", pos:"adj", translation:"двусмысленный", topics:["Advanced Vocabulary"], srs:"learning", streak:3, accuracy:75, hist:[1,1,0,1,0,0,1,1,1,0,0,0,1,1,0,1,1,1,0,1,1,0,0,1,1,1,0,1], defs:[{pos:"adjective",en:"Open to more than one interpretation; not having one obvious meaning",ru:"допускающий двоякое толкование",ex:"\"The professor's ambiguous remarks confused the students.\"",exRu:"Двусмысленные замечания профессора запутали студентов."}], added:"2026-03-09", reviews:12, due:"2d", img:"🌫️", notes:"Often confused with 'vague'. Ambiguous = multiple meanings, vague = unclear meaning.", sources:[{type:"book",title:"Thinking, Fast and Slow",author:"D. Kahneman",context:"Ch.4 — \"The ambiguous nature of our intuitive judgments...\""}] },
  { id:2, word:"audacity", ipa:"/ɔːˈdæs.ɪ.ti/", pos:"noun", translation:"смелость, дерзость", topics:["Advanced Vocabulary","Emotions & Feelings"], srs:"new", streak:0, accuracy:0, hist:[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0], defs:[{pos:"noun",en:"A willingness to take bold risks",ru:"смелость, дерзость",ex:"\"She had the audacity to challenge the CEO's decision.\"",exRu:"У неё хватило дерзости оспорить решение генерального директора."},{pos:"noun",en:"Rude or disrespectful behaviour; impudence",ru:"наглость",ex:"\"He had the audacity to show up uninvited.\"",exRu:"У него хватило наглости прийти без приглашения."}], added:"2026-03-09", reviews:0, due:"now", img:"⚡", notes:"", sources:[{type:"movie",title:"The Social Network",context:"Deposition scene — lawyer uses this word"},{type:"podcast",title:"Lex Fridman #402",context:"Guest used it describing Elon's management style"}] },
  { id:3, word:"benevolent", ipa:"/bəˈnev.əl.ənt/", pos:"adj", translation:"доброжелательный", topics:["Emotions & Feelings"], srs:"review", streak:7, accuracy:88, hist:[1,1,1,0,1,1,1,1,0,1,1,1,1,1,0,1,1,1,1,0,1,1,1,1,1,1,1,1], defs:[{pos:"adjective",en:"Well meaning and kindly; showing goodwill",ru:"доброжелательный, благожелательный",ex:"\"The benevolent king was loved by all his subjects.\"",exRu:"Доброжелательного короля любили все его подданные."}], added:"2026-03-05", reviews:14, due:"5d", img:"🤲", notes:"Latin: bene (well) + volent (wishing). Same root as 'benevolence', 'benefit'.", sources:[{type:"book",title:"The Lord of the Rings",author:"J.R.R. Tolkien",context:"Describing Gandalf's character"},{type:"article",title:"The Guardian",context:"Editorial about philanthropy"}] },
  { id:4, word:"catalyst", ipa:"/ˈkæt.əl.ɪst/", pos:"noun", translation:"катализатор", topics:["Academic English"], srs:"mastered", streak:12, accuracy:95, hist:[1,1,1,1,1,0,1,1,1,1,1,1,1,1,1,1,0,1,1,1,1,1,1,1,1,1,1,1], defs:[{pos:"noun",en:"A person or thing that precipitates an event or change",ru:"катализатор, ускоритель",ex:"\"The discovery was a catalyst for further research.\"",exRu:"Открытие стало катализатором для дальнейших исследований."}], added:"2026-02-20", reviews:22, due:"14d", img:"🔥", notes:"", sources:[] },
  { id:5, word:"conundrum", ipa:"/kəˈnʌn.drəm/", pos:"noun", translation:"головоломка", topics:["Advanced Vocabulary"], srs:"learning", streak:2, accuracy:60, hist:[0,1,0,1,1,0,0,1,0,1,0,0,1,0,1,1,0,0,1,0,1,0,1,1,0,1,0,1], defs:[{pos:"noun",en:"A confusing and difficult problem or question",ru:"загадка, головоломка",ex:"\"The budget deficit remains a conundrum for the government.\"",exRu:"Бюджетный дефицит остаётся головоломкой для правительства."}], added:"2026-03-01", reviews:8, due:"1d", img:"🧩", notes:"", sources:[{type:"article",title:"The Economist",context:"Cover story about climate policy"}] },
  { id:6, word:"eloquent", ipa:"/ˈel.ə.kwənt/", pos:"adj", translation:"красноречивый", topics:["Advanced Vocabulary","Business & Work"], srs:"review", streak:5, accuracy:82, hist:[1,0,1,1,1,0,1,1,0,1,1,1,0,1,1,1,1,0,1,1,1,0,1,1,1,1,1,0], defs:[{pos:"adjective",en:"Fluent or persuasive in speaking or writing",ru:"красноречивый, выразительный",ex:"\"She gave an eloquent speech at the conference.\"",exRu:"Она произнесла красноречивую речь на конференции."}], added:"2026-02-25", reviews:16, due:"3d", img:"🎙️", notes:"", sources:[] },
  { id:7, word:"empathy", ipa:"/ˈem.pə.θi/", pos:"noun", translation:"эмпатия", topics:["Emotions & Feelings","Daily Life"], srs:"mastered", streak:15, accuracy:93, hist:[1,1,1,1,1,1,1,0,1,1,1,1,1,1,1,1,1,1,1,1,1,0,1,1,1,1,1,1], defs:[{pos:"noun",en:"The ability to understand and share the feelings of another",ru:"эмпатия, сопереживание",ex:"\"A good leader shows empathy towards their team.\"",exRu:"Хороший лидер проявляет эмпатию к своей команде."}], added:"2026-02-15", reviews:25, due:"21d", img:"💛", notes:"Empathy ≠ sympathy. Empathy is feeling WITH, sympathy is feeling FOR.", sources:[{type:"book",title:"Nonviolent Communication",author:"M. Rosenberg",context:"Ch.1 — core concept of the book"}] },
  { id:8, word:"enigma", ipa:"/ɪˈnɪɡ.mə/", pos:"noun", translation:"загадка", topics:["Advanced Vocabulary"], srs:"learning", streak:1, accuracy:50, hist:[0,0,1,0,0,1,0,0,0,1,0,0,1,0,0,0,1,0,1,0,0,0,1,0,1,0,0,1], defs:[{pos:"noun",en:"A person or thing that is mysterious or difficult to understand",ru:"загадка, тайна",ex:"\"She remained an enigma to all who knew her.\"",exRu:"Она оставалась загадкой для всех, кто её знал."}], added:"2026-03-07", reviews:4, due:"now", img:"❓", notes:"", sources:[{type:"movie",title:"The Imitation Game",context:"Turing's Enigma machine — the word itself is the title reference"}] },
  { id:9, word:"ephemeral", ipa:"/ɪˈfem.ər.əl/", pos:"adj", translation:"мимолётный", topics:["Advanced Vocabulary","Academic English"], srs:"new", streak:0, accuracy:0, hist:[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0], defs:[{pos:"adjective",en:"Lasting for a very short time",ru:"мимолётный, недолговечный",ex:"\"The ephemeral beauty of cherry blossoms draws millions of visitors.\"",exRu:"Мимолётная красота сакуры привлекает миллионы посетителей."}], added:"2026-03-12", reviews:0, due:"now", img:"🌸", notes:"", sources:[] },
  { id:10, word:"harmony", ipa:"/ˈhɑː.mə.ni/", pos:"noun", translation:"гармония", topics:["Emotions & Feelings","Daily Life"], srs:"mastered", streak:20, accuracy:97, hist:[1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,0,1,1,1,1,1,1,1,1,1], defs:[{pos:"noun",en:"The state of being in agreement or concord",ru:"гармония, согласие",ex:"\"They lived together in perfect harmony.\"",exRu:"Они жили вместе в полной гармонии."}], added:"2026-02-10", reviews:30, due:"30d", img:"☯️", notes:"Greek: harmonia — 'joining'. Used in music, philosophy, and everyday life.", sources:[{type:"book",title:"Sapiens",author:"Y.N. Harari",context:"Chapter on cooperation and social harmony"}] },
  { id:11, word:"inevitable", ipa:"/ɪˈnev.ɪ.tə.bəl/", pos:"adj", translation:"неизбежный", topics:["Advanced Vocabulary","Academic English"], srs:"review", streak:9, accuracy:85, hist:[1,0,1,1,1,1,0,1,1,1,1,1,0,1,1,1,1,1,1,0,1,1,1,1,1,1,1,1], defs:[{pos:"adjective",en:"Certain to happen; unavoidable",ru:"неизбежный, неминуемый",ex:"\"Change is inevitable in any growing organization.\"",exRu:"Перемены неизбежны в любой растущей организации."},{pos:"adjective",en:"So frequently experienced or seen that it is predictable",ru:"привычный, предсказуемый",ex:"\"The inevitable delays at the airport frustrated travelers.\"",exRu:"Привычные задержки в аэропорту раздражали путешественников."}], added:"2026-02-28", reviews:18, due:"4d", img:"⏳", notes:"Latin: in- (not) + evitabilis (avoidable), from evitare (to avoid).\n\nOften confused with 'unavoidable' — they're synonyms, but 'inevitable' carries a stronger sense of destiny/fate, while 'unavoidable' is more practical.\n\nCollocations: inevitable consequence, inevitable outcome, inevitable result, inevitable conclusion.\n\nThanos meme: \"I am inevitable\" — good for remembering the dramatic weight of the word.", sources:[{type:"movie",title:"Avengers: Endgame",context:"Thanos: \"I am inevitable\" — the most iconic use of this word in pop culture"},{type:"book",title:"Sapiens",author:"Y.N. Harari",context:"Ch.12 — discussing the inevitable rise of empires and their eventual collapse"},{type:"podcast",title:"Huberman Lab #127",context:"Andrew discussing inevitable adaptation in neuroplasticity"},{type:"article",title:"The New York Times",context:"Opinion piece: \"The Inevitable Reckoning of Big Tech\" — used in the headline"},{type:"book",title:"The Inevitable",author:"Kevin Kelly",context:"The entire book explores 12 technological forces that are inevitable"}] },
];

/* ─── time grouping ─── */

function getTimeBucket(dateStr) {
  const d = new Date(dateStr);
  const now = new Date("2026-03-25");
  const diff = Math.floor((now - d) / 86400000);

  if (diff <= 7) return { key: "0-week", label: "This week", icon: "◉" };
  if (diff <= 14) return { key: "1-2week", label: "Last week", icon: "○" };
  if (diff <= 31) return { key: "2-month", label: "This month", icon: "◌" };
  
  const months = ["January","February","March","April","May","June","July","August","September","October","November","December"];
  return { key: `3-${d.getMonth()}`, label: months[d.getMonth()], icon: "·" };
}

function groupByTime(ws) {
  const sorted = [...ws].sort((a,b) => new Date(b.added) - new Date(a.added));
  const groups = {};
  const order = [];
  sorted.forEach(w => {
    const b = getTimeBucket(w.added);
    if (!groups[b.key]) { groups[b.key] = { label: b.label, icon: b.icon, words: [] }; order.push(b.key); }
    groups[b.key].words.push(w);
  });
  return order.map(k => groups[k]);
}

function groupByLetter(ws) {
  const grouped = {};
  ws.forEach(w => {
    const l = w.word[0].toUpperCase();
    if (!grouped[l]) grouped[l] = [];
    grouped[l].push(w);
  });
  return Object.keys(grouped).sort().map(l => ({
    label: l,
    words: grouped[l],
    count: grouped[l].length,
  }));
}

/* ─── small components ─── */

function Dots({ hist }) {
  return (
    <div style={{ display:"flex", flexWrap:"wrap", gap:3, width: 9*7-3 }}>
      {hist.map((v,i) => (
        <div key={i} style={{
          width:6, height:6, borderRadius:"50%",
          background: v ? "var(--accent)" : "var(--dot-off)",
          opacity: v ? .35 + (i/hist.length)*.65 : 1,
        }}/>
      ))}
    </div>
  );
}

function TopicDot({ topics }) {
  if (topics.length===1) {
    const c = TOPICS[topics[0]]?.color||"#999";
    return <div style={{width:8,height:8,borderRadius:"50%",background:c,opacity:.65,flexShrink:0}}/>;
  }
  return (
    <div style={{position:"relative",width:14,height:8,flexShrink:0}}>
      {topics.slice(0,2).map((t,i)=>(
        <div key={t} style={{
          position:"absolute",left:i*6,top:0,width:8,height:8,borderRadius:"50%",
          background:TOPICS[t]?.color||"#999",opacity:.65,
          outline:"2px solid var(--island)",
        }}/>
      ))}
    </div>
  );
}

function TopicPill({ name }) {
  const t = TOPICS[name]||{color:"#888",bg:"#eee"};
  return (
    <span style={{
      fontSize:11, fontFamily:"var(--fn)", color:t.color,
      background:t.bg, padding:"3px 10px", borderRadius:20, fontWeight:500,
    }}>{name}</span>
  );
}

/* ─── Word Row ─── */

function WordRow({ w, idx, active, onClick, showDate }) {
  return (
    <div onClick={onClick} style={{
      display:"flex", alignItems:"center", gap:14,
      padding:"12px 16px", cursor:"pointer", borderRadius:10,
      background: active ? "var(--row-on)" : "transparent",
      transition:"background .15s",
    }}
      onMouseEnter={e=>{if(!active) e.currentTarget.style.background="var(--row-on)"}}
      onMouseLeave={e=>{if(!active) e.currentTarget.style.background="transparent"}}
    >
      <span style={{
        fontFamily:"var(--fm)", fontSize:11, color:"var(--t4)",
        minWidth:20, textAlign:"right", fontFeatureSettings:"'tnum'",
      }}>{idx}</span>

      <TopicDot topics={w.topics}/>

      <span style={{
        fontFamily:"var(--fs)", fontWeight:600, fontSize:16.5,
        color:"var(--t1)", letterSpacing:"-.01em",
      }}>{w.word}</span>

      <span style={{
        fontFamily:"var(--fm)", fontSize:12, color:"var(--t4)",
        letterSpacing:".02em",
      }}>{w.ipa}</span>

      <span style={{
        fontFamily:"var(--fn)", fontSize:13.5, color:"var(--t3)",
        fontStyle:"italic", flex:1,
        overflow:"hidden", textOverflow:"ellipsis", whiteSpace:"nowrap",
      }}>{w.translation}</span>

      {/* date shown in newest mode */}
      {showDate && (
        <span style={{
          fontFamily:"var(--fm)", fontSize:10, color:"var(--t4)",
          flexShrink:0, opacity:.7,
        }}>{w.added.slice(5)}</span>
      )}

      <div style={{display:"flex",alignItems:"center",gap:8,flexShrink:0}}>
        {w.due==="now" && (
          <span style={{
            fontSize:10, fontFamily:"var(--fm)", fontWeight:600,
            color:"#191816", background:"#c48b7a",
            padding:"2px 8px", borderRadius:10,
          }}>due</span>
        )}
        <span style={{
          fontSize:10, fontFamily:"var(--fm)", fontWeight:600,
          color: SRS[w.srs].color, textTransform:"uppercase",
          letterSpacing:".05em",
        }}>{SRS[w.srs].label}</span>
      </div>
    </div>
  );
}

const SOURCE_ICONS = {
  book: "📖", movie: "🎬", podcast: "🎧", article: "📰", series: "📺", song: "🎵",
};

/* ─── Expanded Card ─── */

function Card({ w, onClose }) {
  const tc = TOPICS[w.topics[0]]?.color||"#888";
  const [notesOpen, setNotesOpen] = useState(false);
  const [sourcesOpen, setSourcesOpen] = useState(false);
  const [tab, setTab] = useState("word");

  return (
    <div style={{
      borderRadius:18, overflow:"hidden",
      margin:"18px -24px 20px",
      boxShadow:"0 12px 50px var(--card-sh), 0 0 0 1px var(--sh-ring)",
      animation:"pop .25s ease",
      position:"relative", zIndex:2,
    }}>
      {/* Header zone */}
      <div style={{ background:"var(--card-head)", padding:"28px 34px 0" }}>
        <div style={{display:"flex",justifyContent:"space-between",alignItems:"flex-start"}}>
          <div>
            <div style={{display:"flex",alignItems:"baseline",gap:12}}>
              <h2 style={{
                fontFamily:"var(--fs)", fontSize:28, fontWeight:700,
                color:"var(--t1)", margin:0, letterSpacing:"-.02em",
              }}>{w.word}</h2>
              <span style={{fontFamily:"var(--fm)",fontSize:14,color:"var(--t4)"}}>{w.ipa}</span>
              <span style={{
                fontFamily:"var(--fn)",fontSize:11.5,color:"var(--t3)",
                background:"var(--zone)",padding:"3px 10px",borderRadius:20,
              }}>{w.pos}</span>
            </div>
            <div style={{display:"flex",gap:6,marginTop:12,flexWrap:"wrap",alignItems:"center"}}>
              {w.topics.map(t=><TopicPill key={t} name={t}/>)}
              <span style={{fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",marginLeft:4}}>
                added {w.added}
              </span>
            </div>
          </div>
          <button onClick={onClose} style={{
            background:"var(--zone)", border:"none", cursor:"pointer",
            width:30, height:30, borderRadius:"50%", fontSize:13,
            color:"var(--t3)", display:"flex", alignItems:"center",
            justifyContent:"center", transition:"background .15s",
          }}>✕</button>
        </div>

        {/* Tabs — inside header */}
        <div style={{
          display:"flex",gap:0,marginTop:16,
        }}>
          {[
            {id:"word",label:"Word",badge:null},
            {id:"context",label:"Context",badge: (w.sources.length||null) },
            {id:"progress",label:"Progress",badge:null},
          ].map(t=>{
            const isActive = tab===t.id;
            return (
              <button key={t.id} onClick={()=>setTab(t.id)} style={{
                fontFamily:"var(--fm)",fontSize:11,fontWeight:isActive?600:500,
                padding:"8px 16px 10px",border:"none",cursor:"pointer",
                background:"transparent",
                color:isActive?tc:"var(--t4)",
                transition:"all .15s",
                display:"flex",alignItems:"center",gap:6,
                position:"relative",
                borderBottom: isActive ? `2px solid ${tc}` : "2px solid transparent",
              }}>
                {t.label}
                {t.badge && (
                  <span style={{
                    fontSize:9,background:tc+"20",color:tc,
                    padding:"1px 5px",borderRadius:8,fontWeight:700,
                  }}>{t.badge}</span>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Tab content — bento */}
      <div style={{
        background:"var(--card-body)",
        padding:"18px",
      }}>
        {/* ── Word tab ── */}
        {tab==="word" && (
          <div style={{
            display:"grid",gridTemplateColumns:"1fr 200px",
            alignItems:"start",gap:10,
          }}>
            <div style={{display:"flex",flexDirection:"column",gap:10}}>
              {w.defs.map((d,i)=>(
                <div key={i} style={{
                  background:"var(--card-head)",borderRadius:14,
                  padding:"20px 22px",
                }}>
                  <div style={{display:"flex",alignItems:"center",gap:10,marginBottom:10}}>
                    <span style={{
                      fontFamily:"var(--fs)",fontSize:13,fontWeight:700,color:tc,
                      width:22,height:22,borderRadius:"50%",background:tc+"14",
                      display:"flex",alignItems:"center",justifyContent:"center",
                    }}>{i+1}</span>
                    <span style={{
                      fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",
                      textTransform:"uppercase",letterSpacing:".06em",
                    }}>{d.pos}</span>
                  </div>

                  <p style={{
                    fontFamily:"var(--fn)",fontSize:15,color:"var(--t1)",
                    margin:"0 0 4px",lineHeight:1.55,fontWeight:500,paddingLeft:32,
                  }}>{d.en}</p>
                  <p style={{
                    fontFamily:"var(--fn)",fontSize:13,color:tc,
                    margin:"0 0 14px",fontStyle:"italic",paddingLeft:32,opacity:.85,
                  }}>{d.ru}</p>

                  <div style={{
                    background:"var(--zone)",padding:"14px 18px",
                    borderRadius:10,marginLeft:32,
                  }}>
                    <p style={{
                      fontFamily:"var(--fn)",fontSize:13.5,color:"var(--t2)",
                      margin:"0 0 5px",lineHeight:1.55,fontStyle:"italic",
                    }} dangerouslySetInnerHTML={{
                      __html: d.ex.replace(
                        new RegExp(`(${w.word})`,'gi'),
                        `<span style="color:${tc};font-weight:600;font-style:normal">$1</span>`
                      )
                    }}/>
                    <p style={{
                      fontFamily:"var(--fn)",fontSize:12.5,color:"var(--t3)",margin:0,lineHeight:1.5,
                    }}>{d.exRu}</p>
                  </div>
                </div>
              ))}
            </div>

            {/* Image */}
            <div style={{
              background:"var(--card-head)",borderRadius:14,
              padding:"22px 16px",
              display:"flex",alignItems:"center",justifyContent:"center",
              fontSize:48,minHeight:90,
            }}>
              {w.img}
            </div>
          </div>
        )}

        {/* ── Context tab ── */}
        {tab==="context" && (
          <div style={{display:"flex",flexDirection:"column",gap:10}}>
            {/* Sources */}
            <div style={{
              background:"var(--card-head)",borderRadius:14,
              padding:"18px 22px",
            }}>
              <div style={{display:"flex",alignItems:"center",justifyContent:"space-between",marginBottom:12}}>
                <span style={{fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",textTransform:"uppercase",letterSpacing:".1em"}}>Sources</span>
                <span style={{fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)"}}>{w.sources.length}</span>
              </div>

              {w.sources.length > 0 ? (
                <>
                  <div style={{
                    position:"relative",
                    maxHeight: sourcesOpen ? "none" : 160,
                    overflow:"hidden",
                    transition:"max-height .3s ease",
                  }}>
                    <div style={{display:"flex",flexDirection:"column",gap:8}}>
                      {w.sources.map((s,i)=>(
                        <div key={i} style={{
                          background:"var(--zone)",borderRadius:10,
                          padding:"12px 14px",
                          display:"flex",gap:12,alignItems:"flex-start",
                          cursor:"pointer",transition:"background .15s",
                        }}
                          onMouseEnter={e=>e.currentTarget.style.background="var(--card-body)"}
                          onMouseLeave={e=>e.currentTarget.style.background="var(--zone)"}
                        >
                          <span style={{fontSize:18,flexShrink:0,marginTop:1}}>
                            {SOURCE_ICONS[s.type]||"📄"}
                          </span>
                          <div style={{minWidth:0,flex:1}}>
                            <div style={{
                              fontFamily:"var(--fn)",fontSize:13,fontWeight:600,
                              color:"var(--t1)",marginBottom:2,
                            }}>
                              {s.title}
                              {s.author && <span style={{fontWeight:400,color:"var(--t3)"}}> · {s.author}</span>}
                            </div>
                            <div style={{
                              fontFamily:"var(--fn)",fontSize:12,color:"var(--t3)",
                              fontStyle:"italic",lineHeight:1.4,
                            }}>{s.context}</div>
                          </div>
                          <span style={{
                            fontFamily:"var(--fm)",fontSize:11,color:"var(--t4)",
                            flexShrink:0,marginTop:2,
                          }}>→</span>
                        </div>
                      ))}
                    </div>

                    {!sourcesOpen && w.sources.length > 2 && (
                      <div style={{
                        position:"absolute",bottom:0,left:0,right:0,height:60,
                        background:"linear-gradient(transparent, var(--card-head))",
                        pointerEvents:"none",
                      }}/>
                    )}
                  </div>

                  {w.sources.length > 2 && (
                    <div
                      onClick={()=>setSourcesOpen(!sourcesOpen)}
                      style={{
                        marginTop:8,cursor:"pointer",
                        fontFamily:"var(--fm)",fontSize:11,color:tc,
                        display:"flex",alignItems:"center",gap:6,
                        opacity:.8,transition:"opacity .15s",
                      }}
                      onMouseEnter={e=>e.currentTarget.style.opacity="1"}
                      onMouseLeave={e=>e.currentTarget.style.opacity=".8"}
                    >
                      {sourcesOpen ? "Show less" : `Show all ${w.sources.length}`}
                      <span style={{
                        transform:sourcesOpen?"rotate(180deg)":"rotate(0)",
                        transition:"transform .2s",display:"inline-block",fontSize:9,
                      }}>▼</span>
                    </div>
                  )}
                </>
              ) : (
                <div style={{color:"var(--t4)",fontSize:13,fontFamily:"var(--fn)"}}>
                  No sources yet
                </div>
              )}
            </div>

            {/* Notes */}
            <div style={{
              background:"var(--card-head)",borderRadius:14,
              padding:"18px 22px",
            }}>
              <Label>Notes</Label>
              {w.notes ? (
                <div style={{position:"relative"}}>
                  <div style={{
                    maxHeight: notesOpen ? "none" : 72,
                    overflow:"hidden",
                    transition:"max-height .3s ease",
                    fontFamily:"var(--fn)",fontSize:13,color:"var(--t2)",
                    lineHeight:1.6,whiteSpace:"pre-wrap",
                  }}>
                    {w.notes}
                  </div>
                  {!notesOpen && w.notes.length > 120 && (
                    <div style={{
                      position:"absolute",bottom:0,left:0,right:0,height:40,
                      background:"linear-gradient(transparent, var(--card-head))",
                      pointerEvents:"none",
                    }}/>
                  )}
                  {w.notes.length > 120 && (
                    <div
                      onClick={()=>setNotesOpen(!notesOpen)}
                      style={{
                        marginTop:6,cursor:"pointer",
                        fontFamily:"var(--fm)",fontSize:11,color:tc,
                        display:"flex",alignItems:"center",gap:6,
                        opacity:.8,transition:"opacity .15s",
                      }}
                      onMouseEnter={e=>e.currentTarget.style.opacity="1"}
                      onMouseLeave={e=>e.currentTarget.style.opacity=".8"}
                    >
                      {notesOpen ? "Show less" : "Show more"}
                      <span style={{
                        transform:notesOpen?"rotate(180deg)":"rotate(0)",
                        transition:"transform .2s",display:"inline-block",fontSize:9,
                      }}>▼</span>
                    </div>
                  )}
                </div>
              ) : (
                <div style={{color:"var(--t4)",fontSize:13,fontFamily:"var(--fn)",cursor:"text"}}>
                  + Add notes...
                </div>
              )}
            </div>
          </div>
        )}

        {/* ── Progress tab ── */}
        {tab==="progress" && (
          <div style={{display:"flex",flexDirection:"column",gap:10}}>
            {/* Stats row — 4 cards */}
            <div style={{display:"grid",gridTemplateColumns:"1fr 1fr 1fr 1fr",gap:10}}>
              {[
                {value:w.streak, label:"Streak", suffix:"", color:w.streak>5?"#8aaa7e":w.streak>0?"#c4ab6e":"var(--t4)"},
                {value:w.accuracy, label:"Accuracy", suffix:"%", color:w.accuracy>80?"#8aaa7e":w.accuracy>50?"#c4ab6e":"var(--t4)"},
                {value:w.reviews, label:"Reviews", suffix:"", color:"var(--t2)"},
                {value:w.due==="now"?"Now":w.due, label:"Next review", suffix:"", color:w.due==="now"?"#c48b7a":"var(--t2)"},
              ].map((s,i)=>(
                <div key={i} style={{
                  background:"var(--card-head)",borderRadius:14,
                  padding:"16px 14px",textAlign:"center",
                }}>
                  <div style={{
                    fontFamily:"var(--fs)",fontSize:26,fontWeight:700,
                    color:s.color,lineHeight:1,
                  }}>
                    {s.value}{s.suffix && <span style={{fontSize:14,opacity:.7}}>{s.suffix}</span>}
                  </div>
                  <div style={{
                    fontFamily:"var(--fm)",fontSize:9,color:"var(--t4)",
                    textTransform:"uppercase",letterSpacing:".08em",marginTop:6,
                  }}>{s.label}</div>
                </div>
              ))}
            </div>

            {/* Review heatmap — full width, bigger */}
            <div style={{
              background:"var(--card-head)",borderRadius:14,
              padding:"20px 22px",
            }}>
              <div style={{display:"flex",justifyContent:"space-between",alignItems:"center",marginBottom:14}}>
                <Label>Review history</Label>
                <span style={{
                  fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",
                }}>Last 28 days</span>
              </div>

              <div style={{display:"flex",gap:6,alignItems:"flex-end"}}>
                {w.hist.map((v,i)=>{
                  const dayNum = i + 1;
                  const isWeekend = dayNum % 7 === 0 || dayNum % 7 === 6;
                  return (
                    <div key={i} style={{
                      flex:1,display:"flex",flexDirection:"column",
                      alignItems:"center",gap:4,
                    }}>
                      <div style={{
                        width:"100%",
                        height: v ? 28 + Math.round((i / w.hist.length) * 12) : 8,
                        borderRadius:4,
                        background: v ? tc : "var(--zone)",
                        opacity: v ? 0.3 + (i / w.hist.length) * 0.7 : 1,
                        transition:"height .3s ease, opacity .3s ease",
                      }}/>
                      {(i === 0 || i === 6 || i === 13 || i === 20 || i === 27) && (
                        <span style={{
                          fontFamily:"var(--fm)",fontSize:7,color:"var(--t4)",
                          opacity: isWeekend ? .4 : .6,
                        }}>{dayNum}</span>
                      )}
                    </div>
                  );
                })}
              </div>

              <div style={{
                display:"flex",gap:12,marginTop:12,
                fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",
              }}>
                <span style={{display:"flex",alignItems:"center",gap:4}}>
                  <span style={{width:8,height:8,borderRadius:2,background:tc,opacity:.5}}/>
                  correct
                </span>
                <span style={{display:"flex",alignItems:"center",gap:4}}>
                  <span style={{width:8,height:8,borderRadius:2,background:"var(--zone)"}}/>
                  missed
                </span>
                <span style={{marginLeft:"auto"}}>
                  {w.hist.filter(v=>v).length}/{w.hist.length} days active
                </span>
              </div>
            </div>

            {/* SRS Journey */}
            <div style={{
              background:"var(--card-head)",borderRadius:14,
              padding:"20px 22px",
            }}>
              <Label>SRS Stage</Label>
              <div style={{display:"flex",gap:4,alignItems:"center"}}>
                {Object.entries(SRS).map(([k,v],i)=>{
                  const isActive = k===w.srs;
                  const isPast = Object.keys(SRS).indexOf(k) < Object.keys(SRS).indexOf(w.srs);
                  return (
                    <div key={k} style={{flex:1,display:"flex",flexDirection:"column",gap:6}}>
                      <div style={{
                        height:6,borderRadius:10,
                        background: isActive ? v.color : isPast ? v.color+"60" : "var(--zone)",
                        transition:"background .3s",
                      }}/>
                      <div style={{
                        fontFamily:"var(--fm)",fontSize:9,
                        color: isActive ? v.color : isPast ? "var(--t3)" : "var(--t4)",
                        fontWeight: isActive ? 700 : 400,
                        textTransform:"uppercase",letterSpacing:".06em",
                        textAlign:"center",
                      }}>{v.label}</div>
                    </div>
                  );
                })}
              </div>

              <div style={{
                marginTop:14,
                fontFamily:"var(--fn)",fontSize:12,color:"var(--t3)",
                lineHeight:1.5,
              }}>
                {w.srs==="new" && "This word hasn't been reviewed yet. Start a study session to begin learning it."}
                {w.srs==="learning" && `You're actively learning this word. Current streak: ${w.streak} correct answers in a row.`}
                {w.srs==="review" && `This word is in your review cycle. You've answered correctly ${w.accuracy}% of the time.`}
                {w.srs==="mastered" && `Great job! This word is well memorized with ${w.accuracy}% accuracy and a ${w.streak}-day streak.`}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Footer */}
      <div style={{
        display:"flex",justifyContent:"flex-end",gap:6,
        padding:"12px 24px",background:"var(--card-head)",alignItems:"center",
      }}>
        <Kbd>↑</Kbd><Kbd>↓</Kbd>
        <span style={{fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",margin:"0 4px 0 1px"}}>navigate</span>
        <Kbd>Esc</Kbd>
        <span style={{fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",marginLeft:1}}>close</span>
      </div>
    </div>
  );
}

function StatBox({value,label,color}) {
  return (
    <div style={{
      flex:1,background:"var(--zone)",borderRadius:10,
      padding:"10px 8px",textAlign:"center",
    }}>
      <div style={{fontFamily:"var(--fs)",fontSize:22,fontWeight:700,color}}>{value}</div>
      <div style={{fontFamily:"var(--fm)",fontSize:9,color:"var(--t4)",textTransform:"uppercase",letterSpacing:".08em",marginTop:2}}>{label}</div>
    </div>
  );
}

function Label({children}) {
  return (
    <div style={{
      fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",
      textTransform:"uppercase",letterSpacing:".1em",marginBottom:12,
    }}>{children}</div>
  );
}

function Kbd({children}) {
  return (
    <span style={{
      fontFamily:"var(--fm)",fontSize:10,padding:"2px 7px",
      borderRadius:6,background:"var(--zone)",color:"var(--t3)",
    }}>{children}</span>
  );
}

/* ─── Island wrapper ─── */

function Island({ label, count, rightLabel, children }) {
  return (
    <div style={{
      background:"var(--island)",
      borderRadius:14,
      boxShadow:"var(--island-sh)",
      padding:"6px 6px",
      marginBottom:16,
      overflow:"visible",
    }}>
      <div style={{padding:"10px 16px 4px",display:"flex",alignItems:"baseline",justifyContent:"space-between"}}>
        <div>
          <span style={{
            fontFamily:"var(--fs)",fontSize:13,fontWeight:700,
            color:"var(--t4)",
          }}>{label}</span>
          {count != null && (
            <span style={{
              fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",
              marginLeft:8,opacity:.6,
            }}>{count}</span>
          )}
        </div>
        {rightLabel && (
          <span style={{
            fontFamily:"var(--fm)",fontSize:10,color:"var(--t4)",opacity:.5,
          }}>{rightLabel}</span>
        )}
      </div>
      {children}
    </div>
  );
}

/* ─── Main ─── */

export default function DictionaryPage() {
  const [active, setActive] = useState(null);
  const [sort, setSort] = useState("az");

  const azGroups = groupByLetter(words);
  const timeGroups = groupByTime(words);

  const allLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ".split("");
  const usedLetters = azGroups.map(g => g.label);

  const dueN = words.filter(w=>w.due==="now").length;

  let gi = 0;

  return (
    <div style={{fontFamily:"var(--fn)",background:"var(--bg)",minHeight:"100vh",color:"var(--t1)"}}>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=Lora:ital,wght@0,400;0,500;0,600;0,700;1,400&family=JetBrains+Mono:wght@400;500;600&family=Nunito:ital,wght@0,400;0,500;0,600;0,700;1,400&display=swap');

        :root {
          --fs: 'Lora', Georgia, serif;
          --fn: 'Nunito', system-ui, sans-serif;
          --fm: 'JetBrains Mono', monospace;

          --bg: #191816;
          --island: #24221f;
          --card-head: #2c2a26;
          --card-body: #24221f;
          --card-side: #2c2a26;
          --zone: #33302b;
          --row-on: #33302b50;
          --sh: rgba(0,0,0,.28);
          --sh-ring: rgba(123,158,199,.05);
          --card-sh: rgba(0,0,0,.50);

          --t1: #e4ddd2;
          --t2: #b8b0a4;
          --t3: #7e786e;
          --t4: #555048;
          --dot-off: #3a3632;
          --accent: #7b9ec7;

          --island-sh: 0 1px 10px rgba(0,0,0,.22), 0 0 0 1px rgba(123,158,199,.04);
        }

        @keyframes pop {
          from { opacity:0; transform:translateY(-4px) }
          to { opacity:1; transform:translateY(0) }
        }

        *{box-sizing:border-box;margin:0;padding:0}
        ::selection{background:#7b9ec730}
      `}</style>

      <div style={{maxWidth:820,margin:"0 auto",padding:"48px 24px"}}>

        {/* Header */}
        <div style={{display:"flex",justifyContent:"space-between",alignItems:"flex-start",marginBottom:28}}>
          <div>
            <h1 style={{
              fontFamily:"var(--fs)",fontSize:34,fontWeight:700,
              letterSpacing:"-.02em",marginBottom:8,
            }}>Dictionary</h1>
            <div style={{
              display:"flex",gap:8,fontFamily:"var(--fn)",fontSize:13,
              color:"var(--t3)",flexWrap:"wrap",alignItems:"center",
            }}>
              <span><b style={{color:"var(--t2)",fontWeight:600}}>{words.length}</b> words</span>
              <span style={{opacity:.3}}>·</span>
              <span>{words.filter(w=>w.pos==="noun").length} nouns</span>
              <span style={{opacity:.3}}>·</span>
              <span>{words.filter(w=>w.pos==="adj").length} adj</span>
              <span style={{opacity:.3}}>·</span>
              <span>{Object.keys(TOPICS).length} topics</span>
              {dueN>0 && <>
                <span style={{opacity:.3}}>·</span>
                <span style={{
                  color:"#c48b7a",fontWeight:600,
                  background:"#c48b7a14",padding:"2px 10px",borderRadius:10,
                }}>{dueN} due</span>
              </>}
            </div>
          </div>
          <button style={{
            fontFamily:"var(--fn)",fontSize:13,fontWeight:600,
            background:"var(--t1)",color:"var(--bg)",
            border:"none",borderRadius:10,padding:"10px 18px",cursor:"pointer",
          }}>+ Add word</button>
        </div>

        {/* Search + sort */}
        <div style={{display:"flex",gap:10,marginBottom:24}}>
          <div style={{
            flex:1,display:"flex",alignItems:"center",
            background:"var(--island)",borderRadius:12,
            padding:"11px 16px",gap:10,
            boxShadow:"var(--island-sh)",
          }}>
            <span style={{color:"var(--t4)",fontSize:15}}>⌕</span>
            <span style={{fontFamily:"var(--fn)",fontSize:14,color:"var(--t4)"}}>Search words...</span>
          </div>

          <div style={{
            display:"flex",background:"var(--island)",borderRadius:10,
            boxShadow:"var(--island-sh)",overflow:"hidden",
          }}>
            {["A–Z","Newest"].map(m=>(
              <button key={m} style={{
                fontFamily:"var(--fn)",fontSize:12,fontWeight:600,
                padding:"11px 16px",border:"none",cursor:"pointer",borderRadius:0,
                background:(m==="A–Z"&&sort==="az")||(m==="Newest"&&sort==="new")?"var(--t1)":"transparent",
                color:(m==="A–Z"&&sort==="az")||(m==="Newest"&&sort==="new")?"var(--bg)":"var(--t3)",
                transition:"all .15s",
              }} onClick={()=>setSort(m==="A–Z"?"az":"new")}>{m}</button>
            ))}
          </div>
        </div>

        {/* Alphabet bubbles — only in A-Z mode */}
        {sort === "az" && (
          <div style={{display:"flex",gap:4,marginBottom:28,flexWrap:"wrap"}}>
            {allLetters.map(l=>{
              const has=usedLetters.includes(l);
              return (
                <span key={l} style={{
                  fontFamily:"var(--fm)",fontSize:10,fontWeight:600,
                  width:26,height:26,borderRadius:"50%",
                  display:"flex",alignItems:"center",justifyContent:"center",
                  color:has?"var(--t2)":"var(--t4)",
                  opacity:has?1:.2,
                  background:has?"var(--island)":"transparent",
                  boxShadow:has?"var(--island-sh)":"none",
                  cursor:has?"pointer":"default",
                  transition:"all .15s",
                }}>{l}</span>
              );
            })}
          </div>
        )}

        {/* ─── A-Z mode ─── */}
        {sort === "az" && azGroups.map(group => {
          return (
            <Island key={group.label} label={group.label} count={group.count}>
              {group.words.map(w => {
                gi++;
                const isA = active===w.id;
                return (
                  <div key={w.id}>
                    {!isA && <WordRow w={w} idx={gi} active={false}
                      onClick={()=>setActive(w.id)}/>}
                    {isA && <Card w={w} onClose={()=>setActive(null)}/>}
                  </div>
                );
              })}
            </Island>
          );
        })}

        {/* ─── Newest mode ─── */}
        {sort === "new" && timeGroups.map(group => {
          return (
            <Island key={group.label} label={group.label} count={group.words.length}
              rightLabel={group.label === "This week" ? "Mar 19–25" :
                          group.label === "Last week" ? "Mar 12–18" : null}>
              {group.words.map(w => {
                gi++;
                const isA = active===w.id;
                return (
                  <div key={w.id}>
                    {!isA && <WordRow w={w} idx={gi} active={false} showDate
                      onClick={()=>setActive(w.id)}/>}
                    {isA && <Card w={w} onClose={()=>setActive(null)}/>}
                  </div>
                );
              })}
            </Island>
          );
        })}
      </div>
    </div>
  );
}
