# Master Prompt — Reference Catalog Dataset Seeding

## Использование

Копируй промт ниже, подставив номер фазы (1-8), и запускай Claude Code:

```bash
claude --print  # или просто claude
```

Затем вставь промт. Или используй как system prompt / initial message.

---

## Промт

> **Подставь `{N}` на номер фазы (1-8).** Всё остальное — без изменений.

```
Ты выполняешь фазу 1 из плана "Reference Catalog Dataset Seeding" для Go-бэкенда MyEnglish.
Рабочая директория: backend_v4/


Прочитай файл задания фазы:
  docs/plans/2026-02-20-ref-catalog-dataset-seeding/phase-1-migrations-domain.md

В нём — полное описание задачи, список reference documents и критерии верификации.

ПРАВИЛА

Используй sub-agent разработку там, где это возможно

КОНТЕКСТ:
- НЕ читай master plan (plan.md) целиком — всё нужное уже в файле фазы.
- НЕ изучай код, не относящийся к задаче — экономь контекст.
- Если нужен паттерн из другого пакета, прочитай один конкретный файл-пример.

КОД:
- Все команды запускай из backend_v4/.
- Подход к разработке - Test Driven Development

GIT:
- Создавай коммиты после завершения задач

ЗАВЕРШЕНИЕ:
- После прохождения всех проверок — выведи краткий отчёт:
  • Результаты верификации (pass/fail по каждому пункту)
  • Известные ограничения или TODO (если есть)
```

---

## Варианты запуска

### Вариант 1: Интерактивный (рекомендуется)

Запусти Claude Code, вставь промт выше. Можешь уточнять по ходу работы.

### Вариант 2: Headless / CI

```bash
claude -p "$(cat <<'PROMPT'
<вставь промт выше>
PROMPT
)"
```

### Вариант 3: С принудительным plan mode

Если хочешь сначала увидеть план, а потом одобрить:

```bash
claude --plan
```

Затем вставь промт — агент составит план и попросит одобрение перед началом кода.

---

## Между фазами

После завершения каждой фазы:

1. **Проверь отчёт агента** — все ли verification пункты прошли
2. **Сделай коммит** если всё ок:
   ```bash
   cd backend_v4
   git add -A && git commit -m "feat(seeder): phase {N} — <краткое описание>"
   ```
3. **Запусти следующую фазу** новой сессией Claude Code (чистый контекст)

---

## Troubleshooting

| Проблема | Решение |
|----------|---------|
| Агент читает слишком много файлов | Напомни: "Читай только файлы из Reference Documents фазы" |
| Агент ломает существующий код | Напомни: "Не трогай код вне scope фазы. Запусти make test" |
| Агент не может найти паттерн | Укажи конкретный файл-пример из таблицы "Код для изучения" |
| make generate ошибка | Проверь что sqlc.yaml / gqlgen.yml обновлены корректно |
| Агент хочет сделать коммит | Напомни: "Не коммить. Я сам решу" |
| Контекст раздулся | Заверши сессию, открой новую, дай тот же промт — агент продолжит с файлов |



You need to implement phase 8 of the "Reference Catalog Dataset Seeding" plan for the MyEnglish app Go backend.
Working directory: backend_v4/

Read the phase task file:
docs/plans/2026-02-20-ref-catalog-dataset-seeding/phase-8-integration-testing.md

It contains a full description of the task, a list of reference documents, and verification criteria.

RULES

Use sub-agent development where possible.

CONTEXT:
- DO NOT read the master plan (plan.md) in its entirety—everything you need is already in the phase file.
- DO NOT study code unrelated to the task—save context.
- If you need a pattern from another package, read one specific example file.

CODE:
- Run all commands from backend_v4/.
- YOU MUST USE Test Driven Development

GIT:
- Create commits after completing tasks

COMPLETION:
- After passing all checks, output a brief report:
• Verification results (pass/fail for each item)
• Known limitations or TODOs (if any)


я собрал все слова, которые встречаются в сериале south park. Мне нужно, чтобы список этих слов я мог использовать как образец того, какие возможные слова для изучения встречаются в этом сериале. Пройдись по каждому слову в этом списке и подготовь список слов, которые нужно исключить из этого списка (несуществующие слова, опечатки. мусорные слова, технические ошибки парсинга итд) datasets/TV_shows/south_park/south_park_words.txt. Т.е нам нужны слова, убрав которые, останется только список слов, которые можно добавить себе в очередь для заучивания. запускай параллельную обработку этого списка через параллельных агентов
