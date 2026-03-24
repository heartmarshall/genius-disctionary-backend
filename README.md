# genius-disctionary-backend
Context
You are redesigning the Dictionary page of a personal English vocabulary learning app called "MyEnglish". The app is built with React + inline styles. You are given:

A reference JSX file showing the target design as a working prototype

Your task: transform the current implementation into the target design, preserving all data structures and functionality while completely reworking the visual layer.

Design Philosophy
Bento-grid layout — the word card is the centerpiece. It uses a CSS Grid bento approach where information blocks are arranged in a fixed, predictable grid. Each block has a consistent visual language (rounded cells with surfaceInset background), emoji section labels, and compact information density.
Not a flashcard app. This is a power-user vocabulary tool. The design should feel like a modern reference tool (think Notion + Linear + Raycast aesthetics), not a gamified language app.
Desaturated, professional palette. No bright saturated colors. All semantic colors (green, orange, purple, teal) are muted and earthy. The palette uses a cool blue-gray page background with pure white cards for strong elevation contrast.

Color System
Surface Hierarchy (critical — must have visible contrast between layers)

Key rule: POS tags (noun, adj) should be subtle — thin border, textTertiary color, no colored backgrounds. Topic tags use their semantic colors but with desaturated tones. SRS level badges use semantic colors for the dot + text but with light backgrounds.
Page Layout
Sidebar (dark, 224px)

Background: #1B1D23 with #2A2D36 borders
Navigation: Dashboard, Dictionary, Study, Topics, Inbox
Active item: semi-transparent blue highlight, white text
Logo: gradient icon (accent → purple) + "MyEnglish" in serif
User avatar: initials in gradient circle, bottom of sidebar

Main Content Area

Page title: "Dictionary" in Source Serif 4, 26px
Stats line below: "15 words · 7 nouns 8 adj · 5 topics" — plain text, no colored badges
Search toolbar: white card with search icon, text input, segmented sort control (A-Z / Newest)
Word list: grouped by alphabetical headers (sticky, accent-colored letter + horizontal rule)

Word Row (collapsed)

Grid: 28px 1fr auto — index | word + POS + IPA + translation | SRS bar + chevron
Hover: white background + subtle shadow + border appears
Click: row completely replaced by expanded WordCard (no duplication)


Word Card — Bento Layout (the core of the redesign)
Architecture
The card is a white container (14px border-radius, shadow, 10px padding) containing:
┌──────────────────────────────────────┐
│  HERO (full width, surfaceInset bg)  │
├──────────────────────┬───────────────┤
│  📖 Definitions      │  🖼️ Media     │
│  (2/3 width,         ├───────────────┤
│   own bordered cell) │  📊 Progress   │
├────────────┬─────────┤  (widgets     │
│📍Encounters│ ✏️ Notes│   span both   │
│            │         │   rows)       │
└────────────┴─────────┴───────────────┘
  [Expanded Panel — full width, below grid, when active]
CSS Grid: gridTemplateColumns: "1fr 1fr 220px", gridTemplateAreas: "defs defs widgets" "enc notes widgets"
Hero Block

Full width, surfaceInset background, 18px/20px padding
Word: 28px Source Serif 4, bold
Same baseline: IPA (JetBrains Mono, 12.5px, textTertiary) + POS label (11px, textMuted)
Second row: SRS level badge + topic tags (small, desaturated) + "added YYYY-MM-DD"
Actions (right-aligned): Edit icon | Delete icon | divider | Close icon

Edit/Delete/Close are 28×28 icon buttons, textMuted color, hover shows cardBg background
Delete hover color: red
Divider: 1px × 16px vertical line between action groups



Definitions Block (gridArea: "defs")

Bordered cell: cardBg background, 1px border, 16px/18px padding
Section label: "📖 DEFINITIONS" — emoji + uppercase 10px, textMuted
Each sense: numbered (accent color), POS tag, borderLeft (3px, first=accent, rest=border)
Definition: 13.5px, textPrimary
Translation: 12.5px, accent color, fontWeight 500
Examples: surfaceInset rounded blocks with italic English quote + textTertiary Russian
In edit mode: "+ Add sense" button in header, inputs replace text, "×" to remove senses

Media Widget (in right column)

Image: 80px height, cover, rounded top corners. Placeholder: gradient surfaceInset→surfaceFooter with 🖼️
Pronunciation bar: two buttons "🇬🇧 UK" / "🇺🇸 US" docked at bottom, surfaceInset bg, hover→accentLight

Progress Widget (in right column)

Section label: "📊 PROGRESS"
Streak: 🔥 emoji + large number + "streak" label | accuracy % on right — in cardBg rounded row
Mini stats: reviews count + next review date
SRS history bar: flex row of small bars (green=correct, red=incorrect, varying height, fade-in opacity)

Encounters Widget (in right column, or bottom-left if grid allows)

Section label: "📍 ENCOUNTERS (N)"
Preview: max 2 items in compact mode (emoji icon + title + author + detail + date)
"Show all N →" button if more than 2
Empty state: dashed border placeholder "📍 Log encounters"
Data model: { type, icon, title, author, detail, date }
Types: 📚 Book, 🎬 Movie/TV, 🎵 Song/Podcast, 🌐 Web

Notes Widget (bottom row)

Section label: "✏️ NOTES"
Preview: 3-line clamp (CSS -webkit-line-clamp: 3) + "Show more →"
Empty: italic "Add notes..." placeholder
In edit mode: textarea

Expanded Panel (below grid)

Appears when "Show more →" or "Show all N →" is clicked
Full width, surfaceInset background, rounded
Header: emoji + section name + "← Collapse" button
Only one panel open at a time (state: expanded = null | "notes" | "encounters")
Notes: full text with 13px font, 1.7 line-height
Encounters: full-size cards with borders, larger icons, full dates

What NOT to do

No bright saturated colors anywhere — everything is muted and desaturated
No heavy borders on POS tags — they should be the quietest elements
No tabs in the word card — use bento grid with everything visible at once
No footer bar with edit/delete — actions live in the hero header
No emojis in section labels that are too large — keep them at 10-14px
No flex-wrap layouts for the bento grid — positions must be fixed and predictable
Definitions must NOT be full-width — it wastes horizontal space. Use 2/3 width with widgets alongside
When definitions has few entries, the empty space should be filled by the bottom row (encounters + notes), not left blank