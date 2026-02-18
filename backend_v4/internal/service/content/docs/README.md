# Content Service

Сервис управления контентом словарных записей (entries). Управляет четырьмя типами дочерних сущностей: **Sense** (значение), **Translation** (перевод), **Example** (пример) и **UserImage** (пользовательское изображение). Иерархия: Entry → Sense → Translation/Example, Entry → UserImage. Все мутации атомарны (в транзакции) с аудитом.

## Зависимости

```
entryRepo       — проверка ownership записи (GetByID с userID)
senseRepo       — CRUD значений + счётчик + reorder
translationRepo — CRUD переводов + счётчик + reorder
exampleRepo     — CRUD примеров + счётчик + reorder
imageRepo       — CRUD пользовательских изображений + счётчик
auditRepo       — запись аудита для мутаций
txManager       — управление транзакциями
```

Все интерфейсы определены приватно в `service.go`. Wiring через duck typing в `main.go`.

## Проверка ownership

Два паттерна проверки владения:

1. **Прямые дочерние** (Sense, UserImage → Entry): `entryRepo.GetByID(userID, entryID)` или `imageRepo.GetUserByIDForUser(userID, imageID)` — JOIN через entry до user.
2. **Вложенные дочерние** (Translation, Example → Sense → Entry): `senseRepo.GetByIDForUser(userID, senseID)` — один SQL с JOIN по цепочке sense → entry → user. Аналогично для Translation/Example (`GetByIDForUser`).

Все проверки ownership выполняются **внутри транзакции** для предотвращения TOCTOU race conditions.

## Операции

### Sense (значения)

| Метод | Input | Возврат | Транзакция | Аудит |
|-------|-------|---------|------------|-------|
| `AddSense` | `AddSenseInput{EntryID, Definition?, PartOfSpeech?, CEFRLevel?, Translations[]}` | `*domain.Sense` | да | CREATE на Sense |
| `UpdateSense` | `UpdateSenseInput{SenseID, Definition?, PartOfSpeech?, CEFRLevel?}` | `*domain.Sense` | да | UPDATE на Sense (только изменённые поля) |
| `DeleteSense` | `senseID uuid.UUID` | `error` | да | DELETE на Sense |
| `ReorderSenses` | `ReorderSensesInput{EntryID, Items[]}` | `error` | да | нет |

### Translation (переводы)

| Метод | Input | Возврат | Транзакция | Аудит |
|-------|-------|---------|------------|-------|
| `AddTranslation` | `AddTranslationInput{SenseID, Text}` | `*domain.Translation` | да | UPDATE на Sense (translation_added) |
| `UpdateTranslation` | `UpdateTranslationInput{TranslationID, Text}` | `*domain.Translation` | да | UPDATE на Sense (translation_text) |
| `DeleteTranslation` | `translationID uuid.UUID` | `error` | да | UPDATE на Sense (translation_deleted) |
| `ReorderTranslations` | `ReorderTranslationsInput{SenseID, Items[]}` | `error` | да | нет |

### Example (примеры)

| Метод | Input | Возврат | Транзакция | Аудит |
|-------|-------|---------|------------|-------|
| `AddExample` | `AddExampleInput{SenseID, Sentence, Translation?}` | `*domain.Example` | да | UPDATE на Sense (example_added) |
| `UpdateExample` | `UpdateExampleInput{ExampleID, Sentence, Translation?}` | `*domain.Example` | да | UPDATE на Sense (только изменённые поля) |
| `DeleteExample` | `exampleID uuid.UUID` | `error` | да | UPDATE на Sense (example_deleted) |
| `ReorderExamples` | `ReorderExamplesInput{SenseID, Items[]}` | `error` | да | нет |

### UserImage (пользовательские изображения)

| Метод | Input | Возврат | Транзакция | Аудит |
|-------|-------|---------|------------|-------|
| `AddUserImage` | `AddUserImageInput{EntryID, URL, Caption?}` | `*domain.UserImage` | да | UPDATE на Entry (user_image_added) |
| `UpdateUserImage` | `UpdateUserImageInput{ImageID, Caption?}` | `*domain.UserImage` | да | UPDATE на Entry (только изменённые поля) |
| `DeleteUserImage` | `imageID uuid.UUID` | `error` | да | UPDATE на Entry (user_image_deleted) |

**Аудит на родителя**: Translation/Example аудитируются как UPDATE на Sense; UserImage — как UPDATE на Entry. Только Sense аудитируется как самостоятельная сущность (CREATE/UPDATE/DELETE).

## Потоки выполнения

### AddSense

```
UserID из ctx → Validate → RunInTx {
    entryRepo.GetByID(userID, entryID)     // ownership check
    senses.CountByEntry(entryID) >= 20?    // лимит
    senses.CreateCustom(entryID, ..., "user")
    for each translation:
        trim → translations.CreateCustom(senseID, text, "user")
    audit.Log(CREATE на Sense)
} → log DEBUG → return sense
```

### UpdateSense

```
UserID из ctx → Validate → RunInTx {
    old = senses.GetByIDForUser(userID, senseID)  // ownership через JOIN
    updated = senses.Update(senseID, ...)
    changes = buildSenseChanges(old, input)        // diff по Definition, PartOfSpeech, CEFRLevel
    if changes не пуст → audit.Log(UPDATE на Sense)
} → return updated
```

No-op update (все поля идентичны) не создаёт запись аудита.

### DeleteSense

```
UserID из ctx → RunInTx {
    sense = senses.GetByIDForUser(userID, senseID)  // ownership
    senses.Delete(senseID)                           // CASCADE: translations + examples
    log INFO → audit.Log(DELETE на Sense)
}
```

### ReorderSenses

```
UserID из ctx → Validate → RunInTx {
    entryRepo.GetByID(userID, entryID)       // ownership
    existing = senses.GetByEntryID(entryID)  // проверка принадлежности items
    for each item: item.ID in existing?      // если нет → ValidationError
    senses.Reorder(items)
}
```

### AddTranslation

```
UserID из ctx → Validate → trim text → RunInTx {
    senses.GetByIDForUser(userID, senseID)         // ownership через JOIN
    translations.CountBySense(senseID) >= 20?      // лимит
    translations.CreateCustom(senseID, text, "user")
    audit.Log(UPDATE на Sense: translation_added)
} → log DEBUG → return translation
```

### UpdateTranslation

```
UserID из ctx → Validate → trim text → RunInTx {
    old = translations.GetByIDForUser(userID, translationID)  // ownership через JOIN
    updated = translations.Update(translationID, text)
    audit.Log(UPDATE на Sense: translation_text old→new)
} → return updated
```

### DeleteTranslation

```
UserID из ctx → RunInTx {
    translation = translations.GetByIDForUser(userID, translationID)  // ownership
    translations.Delete(translationID)
    log INFO → audit.Log(UPDATE на Sense: translation_deleted)
}
```

### ReorderTranslations

```
UserID из ctx → Validate → RunInTx {
    senses.GetByIDForUser(userID, senseID)                // ownership
    existing = translations.GetBySenseID(senseID)         // проверка принадлежности
    for each item: item.ID in existing?
    translations.Reorder(items)
}
```

### AddExample

```
UserID из ctx → Validate → trim sentence + translation → RunInTx {
    senses.GetByIDForUser(userID, senseID)       // ownership через JOIN
    examples.CountBySense(senseID) >= 50?        // лимит
    examples.CreateCustom(senseID, sentence, translation, "user")
    audit.Log(UPDATE на Sense: example_added)
} → log DEBUG → return example
```

### UpdateExample

```
UserID из ctx → Validate → trim sentence + translation → RunInTx {
    old = examples.GetByIDForUser(userID, exampleID)  // ownership через JOIN
    updated = examples.Update(exampleID, sentence, translation)
    changes = buildExampleChanges(old, sentence, translation)
    if changes не пуст → audit.Log(UPDATE на Sense)
} → return updated
```

`translation=nil` означает удаление перевода (NULL в БД).

### DeleteExample

```
UserID из ctx → RunInTx {
    example = examples.GetByIDForUser(userID, exampleID)  // ownership
    examples.Delete(exampleID)
    log INFO → audit.Log(UPDATE на Sense: example_deleted)
}
```

### ReorderExamples

```
UserID из ctx → Validate → RunInTx {
    senses.GetByIDForUser(userID, senseID)         // ownership
    existing = examples.GetBySenseID(senseID)      // проверка принадлежности
    for each item: item.ID in existing?
    examples.Reorder(items)
}
```

### AddUserImage

```
UserID из ctx → Validate → trim URL → RunInTx {
    entryRepo.GetByID(userID, entryID)          // ownership
    images.CountUserByEntry(entryID) >= 20?     // лимит
    images.CreateUser(entryID, url, caption)
    audit.Log(UPDATE на Entry: user_image_added)
} → log DEBUG → return image
```

### UpdateUserImage

```
UserID из ctx → Validate → RunInTx {
    old = images.GetUserByIDForUser(userID, imageID)  // ownership через JOIN
    updated = images.UpdateUser(imageID, caption)
    changes = buildImageCaptionChanges(old.Caption, input.Caption)
    if changes не пуст → audit.Log(UPDATE на Entry)
} → return updated
```

### DeleteUserImage

```
UserID из ctx → RunInTx {
    image = images.GetUserByIDForUser(userID, imageID)  // ownership через JOIN
    images.DeleteUser(imageID)
    log INFO → audit.Log(UPDATE на Entry: user_image_deleted)
}
```

## Валидация

Каждый Input имеет `Validate() error`, возвращающий `*domain.ValidationError` со всеми ошибками разом.

### AddSenseInput

| Поле | Правило |
|------|---------|
| `entry_id` | обязательное, не uuid.Nil |
| `definition` | опциональное, <= 2000 символов |
| `part_of_speech` | опциональное, `IsValid()` |
| `cefr_level` | опциональное, одно из A1/A2/B1/B2/C1/C2 |
| `translations` | <= 20 элементов; каждый: trimmed, непустой, <= 500 символов |

### UpdateSenseInput

| Поле | Правило |
|------|---------|
| `sense_id` | обязательное, не uuid.Nil |
| `definition` | опциональное, <= 2000 символов |
| `part_of_speech` | опциональное, `IsValid()` |
| `cefr_level` | опциональное, одно из A1/A2/B1/B2/C1/C2 |

### ReorderSensesInput / ReorderTranslationsInput / ReorderExamplesInput

| Поле | Правило |
|------|---------|
| `entry_id` / `sense_id` | обязательное, не uuid.Nil |
| `items` | 1..50 элементов, без дубликатов ID, position >= 0 |

### AddTranslationInput / UpdateTranslationInput

| Поле | Правило |
|------|---------|
| `sense_id` / `translation_id` | обязательное, не uuid.Nil |
| `text` | обязательное, trimmed, <= 500 символов |

### AddExampleInput / UpdateExampleInput

| Поле | Правило |
|------|---------|
| `sense_id` / `example_id` | обязательное, не uuid.Nil |
| `sentence` | обязательное, trimmed, <= 2000 символов |
| `translation` | опциональное, <= 2000 символов |

### AddUserImageInput

| Поле | Правило |
|------|---------|
| `entry_id` | обязательное, не uuid.Nil |
| `url` | обязательное, trimmed, <= 2000, валидный HTTP(S) URL |
| `caption` | опциональное, <= 500 символов |

### UpdateUserImageInput

| Поле | Правило |
|------|---------|
| `image_id` | обязательное, не uuid.Nil |
| `caption` | опциональное, <= 500 символов |

## Лимиты

| Ресурс | Лимит |
|--------|-------|
| Senses на entry | 20 (проверяется в AddSense внутри транзакции) |
| Translations на sense | 20 (проверяется в AddTranslation внутри транзакции) |
| Examples на sense | 50 (проверяется в AddExample внутри транзакции) |
| UserImages на entry | 20 (проверяется в AddUserImage внутри транзакции) |
| Items в reorder | 50 |
| Длина definition | 2000 символов |
| Длина text (translation) | 500 символов |
| Длина sentence/translation (example) | 2000 символов |
| Длина URL (image) | 2000 символов |
| Длина caption (image) | 500 символов |

## Ошибки

Все операции: `ErrUnauthorized` если нет userID в контексте.

| Ситуация | Ошибка |
|----------|--------|
| Невалидный input | `*ValidationError` |
| Entry не найден / чужой / soft-deleted | `ErrNotFound` |
| Sense не найден / чужой | `ErrNotFound` |
| Translation не найден / чужой | `ErrNotFound` |
| Example не найден / чужой | `ErrNotFound` |
| UserImage не найден / чужой | `ErrNotFound` |
| Лимит сущностей | `*ValidationError` |
| Item не принадлежит родителю (reorder) | `*ValidationError` |
| No-op update | не ошибка (аудит пропускается) |

## Файлы

```
service.go           — интерфейсы (7 шт.), конструктор, константы (лимиты, CEFR)
input.go             — 10 Input-структур с Validate(), helpers (isValidHTTPURL, fieldIndex)
sense.go             — AddSense, UpdateSense, DeleteSense, ReorderSenses, buildSenseChanges
translation.go       — Add/Update/Delete/Reorder Translation + Example (8 методов), buildExampleChanges
userimage.go         — AddUserImage, UpdateUserImage, DeleteUserImage, buildImageCaptionChanges
sense_test.go        — тесты Sense CRUD + shared mocks (mockEntryRepo, mockSenseRepo, mockTranslationRepo, mockAuditRepo, mockTxManager)
translation_test.go  — тесты Translation CRUD + Reorder
example_test.go      — тесты Example CRUD + Reorder
userimage_test.go    — тесты UserImage CRUD (limits, update, no-op audit skip)
```

## Тесты

58 unit-тестов, все с `t.Parallel()`. Покрытие: happy path, валидация, ownership, ошибки, аудит, edge cases (no-op update skip audit, image limit, CASCADE delete, reorder item validation, JOIN-based ownership).

```bash
go test -v -race ./internal/service/content/...
```
