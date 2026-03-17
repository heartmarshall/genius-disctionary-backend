# MyEnglish — Design System

## Стек
Backend: Go — микросервисы, производительность.
Frontend: React + TypeScript, Vite, React Router.
Стили: Tailwind CSS с кастомными токенами.
UI-компоненты: shadcn/ui.
Иконки: Lucide — outline 20–24px. Активное состояние — subtle увеличение strokeWidth (2 vs 1.75), не filled.
Анимации: Framer Motion.
Command Palette: cmdk.
Celebrations: canvas-confetti для milestone-анимаций.
i18n: react-i18next + i18next + i18next-browser-languagedetector.

---

## Типографика

Четыре шрифта.

Neue Montreal — body, UI, кнопки, навигация, заголовки. Начертания: 400, 500, 700. Fallback: system-ui, sans-serif.
Orelega One — слово/лемма на карточке (hero-элемент). Начертание: 400. Fallback: Georgia, serif.
EB Garamond — цитаты из книг, IPA-транскрипция. Начертания: 400, 500; italic 400. Fallback: Georgia, serif.
Courier Prime — цитаты из субтитров (TV/кино). Начертание: 400. Fallback: ui-monospace, monospace.

Шкала размеров:
- h1 (page title): text-3xl, weight 700, Neue Montreal
- h2 (section title): text-2xl, weight 700, Neue Montreal
- h3 (card title): text-xl, weight 500, Neue Montreal
- body: text-base, weight 400, Neue Montreal
- small / meta: text-sm, weight 400, Neue Montreal
- caption / label: text-xs, weight 500, Neue Montreal
- word hero: text-4xl, weight 400, Orelega One
- IPA / цитата книги: text-base, weight 400, EB Garamond
- цитата субтитров: text-sm, weight 400, Courier Prime

---

## Цветовая палитра — Herbarium

Три непересекающихся слоя. Коричневые тона полностью исключены.

### Слой 1 — Интерфейс (нейтрали)
- `--bg-page` #ffffff — фон страницы
- `--bg-card` #ffffff — фон карточек
- `--surface-secondary` #f8f8f8 — sidebar, панели
- `--surface-disabled` #e8e8e8 — disabled, skeleton
- `--border-default` #d4d4d4 — рамки карточек, инпуты
- `--border-subtle` #e8e8e8 — разделители
- `--text-primary` #1a1a1a — заголовки, основной текст
- `--text-secondary` #666666 — мета, подписи
- `--text-tertiary` #999999 — placeholder
- `--text-disabled` #b8b8b8 — disabled
- `--text-on-accent` #ffffff — текст на цветных кнопках

### Слой 2 — Функциональный 
SRS-кнопки, статусы слов, уведомления. Каждый цвет = base + hover + light + fg.

Poppy — Again / Error / Danger / New. Base `oklch(0.52 0.10 35)` #a14832, hover #8a3b28, light #f5ebe7, fg #7d3e2b.
Goldenrod — Hard / Warning / Learning. Base `oklch(0.56 0.08 85)` #b9983d, hover #766536, light #f3f0e5, fg #6b5e34.
Cornflower — Good / Info / Review. Base `oklch(0.50 0.06 250)` #55698a, hover #475a78, light #e8ecf2, fg #3f526d.
Thyme — Easy / Success / Mastered. Base `oklch(0.48 0.06 140)` #4c7458, hover #40654b, light #eff5f0, fg #3a6146.

Семантические алиасы: `--accent` = Poppy, `--srs-again` = Poppy, `--srs-hard` = Goldenrod, `--srs-good` = Cornflower, `--srs-easy` = Thyme, `--danger` = Poppy, `--success` = Thyme, `--warning` = Goldenrod.

### Слой 3 — Контент (источники)
Книга/статья — badge #7a6898 (lavender), light bg #f0ecf5, шрифт EB Garamond.
Экран (TV/кино) — badge #4e7385, light bg #e8eff3, шрифт Courier Prime.
Музыка/подкаст — badge #8a6482 (heather), light bg #f3ecf1, шрифт EB Garamond italic.

---

## Spacing и радиусы

Spacing — стандартная Tailwind-шкала (1 unit = 4px). Кастомные значения не добавлять.
Border-radius — стандартная Tailwind-шкала (rounded-sm, rounded-md, rounded-lg и т.д.). Кастомные значения не добавлять.

---

## Tailwind-конфиг

Все цветовые токены подключаются через CSS-переменные в tailwind.config.ts:

```ts
theme: {
  extend: {
    colors: {
      'bg-page': 'var(--bg-page)',
      'bg-card': 'var(--bg-card)',
      'surface-secondary': 'var(--surface-secondary)',
      'surface-disabled': 'var(--surface-disabled)',
      'border-default': 'var(--border-default)',
      'border-subtle': 'var(--border-subtle)',
      'text-primary': 'var(--text-primary)',
      'text-secondary': 'var(--text-secondary)',
      'text-tertiary': 'var(--text-tertiary)',
      'text-disabled': 'var(--text-disabled)',
      'text-on-accent': 'var(--text-on-accent)',
      poppy: {
        DEFAULT: 'var(--poppy)',
        hover: 'var(--poppy-hover)',
        light: 'var(--poppy-light)',
        fg: 'var(--poppy-fg)',
      },
      goldenrod: {
        DEFAULT: 'var(--goldenrod)',
        hover: 'var(--goldenrod-hover)',
        light: 'var(--goldenrod-light)',
        fg: 'var(--goldenrod-fg)',
      },
      cornflower: {
        DEFAULT: 'var(--cornflower)',
        hover: 'var(--cornflower-hover)',
        light: 'var(--cornflower-light)',
        fg: 'var(--cornflower-fg)',
      },
      thyme: {
        DEFAULT: 'var(--thyme)',
        hover: 'var(--thyme-hover)',
        light: 'var(--thyme-light)',
        fg: 'var(--thyme-fg)',
      },
    },
  },
}
```

---

## Layout и breakpoints

Desktop-first. Breakpoints — стандартные Tailwind (sm 640px, md 768px, lg 1024px, xl 1280px). Кастомные не добавлять.

Максимальная ширина контейнера: max-w-7xl.
На мобильном (< md): sidebar — overlay, основной контент — full width.

Словарь поддерживает два режима отображения: grid (grid-cols-2 lg:grid-cols-3) и list.

---

## Файловая структура

```
src/
  components/
    ui/          # shadcn-компоненты, не редактировать вручную
    common/      # переиспользуемые компоненты
    [feature]/   # компоненты конкретной фичи (dictionary/, study/, etc.)
  pages/         # page-компоненты, один файл = один роут
  hooks/         # кастомные хуки
  lib/           # утилиты, helpers, константы
  types/         # TypeScript-интерфейсы и типы
  graphql/       # запросы и мутации
  router/        # роутинг
  assets/        # шрифты, статика
```

Нейминг:
- Компоненты: PascalCase (WordCard.tsx)
- Хуки: camelCase с префиксом use (useWordCard.ts)
- Утилиты: camelCase (formatDate.ts)
- Типы: PascalCase (WordEntry, StudySession)

---

## UI-библиотеки и компоненты

shadcn/ui — единственный источник UI-компонентов. Перед созданием кастомного компонента проверить, есть ли подходящий в shadcn. Copy-paste модель: компоненты живут в проекте, полный контроль. Кастомизация через Tailwind-токены.

Lucide — единственный набор иконок, без исключений. Outline 20–24px, strokeWidth 1.75 по умолчанию. Активное состояние в навигации — strokeWidth 2 (без filled). Каждая иконка в sidebar имеет уникальную CSS hover-анимацию (rotate, translate, scale) через group-hover + transition-transform duration-200.

---

## Анимации

CSS transition — для простых состояний (hover, focus, disabled): duration-150 или duration-200.
Per-icon hover-анимации в sidebar — каждая иконка со своей уникальной CSS-анимацией через group-hover + transition-transform duration-200: Settings rotate-90, BookOpen -rotate-6 + scale-105, GraduationCap -translate-y-0.5 + rotate-3, Tags rotate-12, Inbox -translate-y-0.5, LogOut translate-x-0.5, Dashboard/Shield scale-110.
Framer Motion — только для появления/исчезновения элементов, SRS-фидбека и page transitions.

---

## Ключевые дизайн-решения

### Визуальная иерархия
Иерархию создают типографика и пространство, не декоративные эффекты. Нет теней на карточках, нет градиентов, нет иллюстраций (кроме явно оговорённых).
Разделение блоков — один приём на выбор, не комбинировать: пространство (gap ≥ 6), линия, или контраст фона (surface-secondary).

### Карточки
Фон bg-card, рамка 1px border-default, радиус rounded-md, паддинг p-4. Никаких box-shadow.

### Акцентный цвет (Poppy)
Используется ТОЛЬКО на: primary buttons (bg), active nav (text + icon), progress bars (fill), текстовые ссылки, бейджи/счётчики. Всё остальное — нейтральная палитра.

### Навигация
Sidebar: expanded (w-60) / collapsed (w-16). Toggle: chevron. Ширина анимируется через transition-[width] duration-300. Active item: poppy-light bg + poppy text + strokeWidth 2 (без filled, без indicator bar). Текст коллапсируется через max-width transition (не width — CSS не анимирует auto). Tooltips в collapsed-режиме: всегда смонтированы, но скрыты через `open={collapsed ? undefined : false}` на `<Tooltip>` (предотвращает flash при коллапсе). Nav: overflow-y-auto для масштабируемости. Mobile (< md): overlay, backdrop opacity 0.4, Framer Motion slide + fade.
Command Palette (⌘K): секции Navigation → Words (fuzzy) → Actions, max 8 результатов с прокруткой.

### Состояния компонентов
Loading — skeleton с bg-surface-disabled и анимацией pulse. Никаких спиннеров на уровне страницы.
Empty — текст + опциональный CTA. Без иллюстраций.
Error — инлайн: текст с причиной и действием. Toast для глобальных ошибок.
Success — toast для глобальных действий (сохранено, удалено).

### Формы и инпуты
Состояния инпута:
- Default: border-default
- Focus: ring cornflower
- Error: border poppy
- Disabled: border-subtle, text-disabled

Лейбл — над инпутом, text-sm, weight 500.
Сообщение об ошибке — под инпутом, text-xs, цвет poppy-fg.
Required — звёздочка * после лейбла, цвет poppy.

### Z-index
- Base content: 0
- Dropdown / Popover: z-10
- Sidebar overlay: z-20
- Modal: z-30
- Toast: z-40
- Command Palette: z-50

### Тон интерфейса
Обращение на «вы», без сленга и панибратства. Emoji только в celebrations, не в основном UI. Числа вместо оценок: «Осталось 5 слов», не «Почти готово!». Ошибки = проблема + решение: «Не сохранено. Проверьте соединение».

---

## Доступность

- Все интерактивные элементы доступны с клавиатуры.
- Focus ring всегда видим — не убирать outline-none без замены.
- Иконки без текста имеют aria-label.
- `aria-hidden` — использовать `aria-hidden={value || undefined}`, не `aria-hidden={value}`. Избегать `aria-hidden="false"` — лучше отсутствие атрибута.
- Контраст текста соответствует WCAG AA.

---

## i18n (интернационализация)

Стек: react-i18next + i18next + i18next-browser-languagedetector.
Языки: en (fallback), ru. Определение — автоматическое из `navigator.language`, сохранение в `localStorage`.

Структура переводов:
```
src/i18n/
  index.ts
  locales/
    en/<namespace>.json
    ru/<namespace>.json
```

Namespace-ы соответствуют фичам: `auth`, `validation`, далее по мере роста (`dictionary`, `study`, etc.).
Ключи — flat, через точку: `login.title`, `register.submit`, `field.email`.

Переключатель языка на auth-страницах — кнопка EN/RU в правом верхнем углу `AuthLayout`. Стиль: `text-sm text-text-secondary hover:text-poppy`.

Хардкод строк в компонентах запрещён для экранов, покрытых i18n. Для непокрытых экранов — допустим до момента перевода.

---

## Запрещено

- Коричневые/umber-тона где угодно
- CSS-in-JS (styled-components, emotion) — конфликт с Tailwind
- Наборы иконок кроме Lucide
- UI-библиотеки поверх shadcn (Material UI, Ant Design)
- Box-shadow на карточках
- Emoji в основном UI
- Сообщения «что-то пошло не так» без указания причины и действия
- Hardcoded hex-значения вне CSS-переменных
- Произвольные spacing-значения вне Tailwind-шкалы
- Кастомные breakpoints

---

## Чеклист для агента

Перед выдачей кода проверить:
- Все цвета используются через Tailwind-токены, нет hardcoded hex
- Все отступы из стандартной Tailwind-шкалы, нет произвольных значений
- Шрифты только из четырёх определённых, с правильным назначением
- Иконки только Lucide
- Нет box-shadow на карточках
- Нет CSS-in-JS
- Framer Motion используется только там, где это оговорено
