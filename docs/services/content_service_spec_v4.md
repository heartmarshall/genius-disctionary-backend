# MyEnglish Backend v4 — Content Service Specification

> **Статус:** Draft v1.0
> **Дата:** 2026-02-12
> **Зависимости:** code_conventions_v4.md (секции 2–4, 6), data_model_v4.md (секция 4), repo_layer_spec_v4.md (секции 5, 12, 19.3–19.4), service_layer_spec_v4.md, business_scenarios_v4.md (C1–C12)

---

## 1. Ответственность

Content Service отвечает за **CRUD дочернего контента** словарных записей: senses (значения), translations (переводы), examples (примеры), user images (пользовательские изображения). Также управляет порядком отображения (reorder).

Content Service **не** отвечает за: создание/удаление entries (DictionaryService), начальное заполнение контента при добавлении слова из каталога (DictionaryService), pronunciations и catalog images (привязываются автоматически при создании entry, API для ручного link/unlink на MVP нет).

### 1.1. Ownership chain

Все операции Content Service требуют проверки ownership: дочерний элемент → entry → user. Это 2–3 запроса на каждую мутацию:

```
Translation → Sense → Entry → User
Example     → Sense → Entry → User
Sense       → Entry → User
UserImage   → Entry → User
```

Для операций, принимающих parent ID (AddSense по entryID, AddTranslation по senseID), ownership проверяется через parent chain. Для операций, принимающих ID самого элемента (UpdateSense по senseID), сначала загружается элемент, затем проверяется его parent chain.

---

## 2. Зависимости

### 2.1. Интерфейсы репозиториев

```
entryRepo interface {
    GetByID(ctx, userID, entryID uuid.UUID) → (*domain.Entry, error)
}

senseRepo interface {
    GetByID(ctx, senseID uuid.UUID) → (*domain.Sense, error)
    GetByEntryID(ctx, entryID uuid.UUID) → ([]domain.Sense, error)
    CountByEntry(ctx, entryID uuid.UUID) → (int, error)
    CreateCustom(ctx, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) → (*domain.Sense, error)
    Update(ctx, senseID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string) → (*domain.Sense, error)
    Delete(ctx, senseID uuid.UUID) → error
    Reorder(ctx, items []ReorderItem) → error
}

translationRepo interface {
    GetByID(ctx, translationID uuid.UUID) → (*domain.Translation, error)
    GetBySenseID(ctx, senseID uuid.UUID) → ([]domain.Translation, error)
    CountBySense(ctx, senseID uuid.UUID) → (int, error)
    CreateCustom(ctx, senseID uuid.UUID, text string, sourceSlug string) → (*domain.Translation, error)
    Update(ctx, translationID uuid.UUID, text string) → (*domain.Translation, error)
    Delete(ctx, translationID uuid.UUID) → error
    Reorder(ctx, items []ReorderItem) → error
}

exampleRepo interface {
    GetByID(ctx, exampleID uuid.UUID) → (*domain.Example, error)
    GetBySenseID(ctx, senseID uuid.UUID) → ([]domain.Example, error)
    CountBySense(ctx, senseID uuid.UUID) → (int, error)
    CreateCustom(ctx, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) → (*domain.Example, error)
    Update(ctx, exampleID uuid.UUID, sentence string, translation *string) → (*domain.Example, error)
    Delete(ctx, exampleID uuid.UUID) → error
    Reorder(ctx, items []ReorderItem) → error
}

imageRepo interface {
    GetUserByID(ctx, imageID uuid.UUID) → (*domain.UserImage, error)
    CreateUser(ctx, entryID uuid.UUID, url string, caption *string) → (*domain.UserImage, error)
    DeleteUser(ctx, imageID uuid.UUID) → error
}

auditRepo interface {
    Log(ctx, record domain.AuditRecord) → error
}
```

ReorderItem: `struct { ID uuid.UUID; Position int }`

### 2.2. Интерфейс TxManager

```
txManager interface {
    RunInTx(ctx, fn func(ctx) error) → error
}
```

### 2.3. Конструктор

```
func NewService(
    logger          *slog.Logger,
    entryRepo       entryRepo,
    senseRepo       senseRepo,
    translationRepo translationRepo,
    exampleRepo     exampleRepo,
    imageRepo       imageRepo,
    auditRepo       auditRepo,
    txManager       txManager,
) *Service
```

Нет конфигурации — лимиты захардкожены как константы:

```
const (
    MaxSensesPerEntry       = 20
    MaxTranslationsPerSense = 20
    MaxExamplesPerSense     = 50
)
```

---

## 3. Ownership Check — внутренний helper

Множество операций требуют проверки ownership. Чтобы не дублировать код, сервис использует приватные helper-методы:

```
// checkEntryOwnership: проверить что entry принадлежит пользователю.
// Возвращает entry для использования в audit/логике.
func (s *Service) checkEntryOwnership(ctx, userID, entryID) → (*domain.Entry, error)
    entry, err = entryRepo.GetByID(ctx, userID, entryID)
    └─ ErrNotFound → return ErrNotFound
    return entry, nil

// checkSenseOwnership: загрузить sense, затем проверить ownership его entry.
// Возвращает sense и entry.
func (s *Service) checkSenseOwnership(ctx, userID, senseID) → (*domain.Sense, *domain.Entry, error)
    sense, err = senseRepo.GetByID(ctx, senseID)
    └─ ErrNotFound → return ErrNotFound
    entry, err = s.checkEntryOwnership(ctx, userID, sense.EntryID)
    └─ ErrNotFound → return ErrNotFound (sense существует, но entry чужой/deleted)
    return sense, entry, nil

// checkTranslationOwnership: загрузить translation → sense → entry → user.
func (s *Service) checkTranslationOwnership(ctx, userID, translationID) → (*domain.Translation, *domain.Entry, error)
    translation, err = translationRepo.GetByID(ctx, translationID)
    └─ ErrNotFound → return ErrNotFound
    _, entry, err = s.checkSenseOwnership(ctx, userID, translation.SenseID)
    return translation, entry, err

// checkExampleOwnership: аналогично checkTranslationOwnership.
func (s *Service) checkExampleOwnership(ctx, userID, exampleID) → (*domain.Example, *domain.Entry, error)

// checkUserImageOwnership: загрузить image → entry → user.
func (s *Service) checkUserImageOwnership(ctx, userID, imageID) → (*domain.UserImage, *domain.Entry, error)
    image, err = imageRepo.GetUserByID(ctx, imageID)
    └─ ErrNotFound → return ErrNotFound
    entry, err = s.checkEntryOwnership(ctx, userID, image.EntryID)
    return image, entry, err
```

Это 2–3 SQL-запроса на каждую мутацию. Для MVP допустимо. Оптимизация (join в repo или кеширование entry в рамках запроса) — post-MVP.

---

## 4. Бизнес-сценарии и операции

### 4.1. AddSense (C1)

**Сценарий:** Пользователь добавляет новое значение к слову (definition, pos, cefr, с переводами).

**Метод:** `AddSense(ctx context.Context, input AddSenseInput) → (*domain.Sense, error)`

**AddSenseInput:**
- EntryID (uuid.UUID)
- Definition (*string)
- PartOfSpeech (*domain.PartOfSpeech)
- CEFRLevel (*string)
- Translations ([]string) — начальные переводы (convenience, чтобы не делать 3 вызова)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. entry, err = s.checkEntryOwnership(ctx, userID, input.EntryID)
   └─ ErrNotFound → return ErrNotFound

4. count, err = senseRepo.CountByEntry(ctx, input.EntryID)
   └─ count >= MaxSensesPerEntry → return ValidationError("senses", "limit reached (20)")

5. txManager.RunInTx(ctx, func(ctx) error {

   5a. sense, err = senseRepo.CreateCustom(ctx, input.EntryID, input.Definition, input.PartOfSpeech, input.CEFRLevel, "user")
       // position auto-increment в repo: MAX(position)+1

   5b. Для каждого text в input.Translations:
       translationRepo.CreateCustom(ctx, sense.ID, text, "user")

   5c. auditRepo.Log(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &sense.ID,
           Action:     domain.ActionCreate,
           Changes: map[string]any{
               "entry_id":          {"new": input.EntryID},
               "definition":        {"new": input.Definition},
               "translations_count": {"new": len(input.Translations)},
           },
       })

   return nil
   })

6. return sense, nil
```

**Corner cases:**
- **Sense без definition:** Definition = nil — допустимо. Sense может содержать только переводы.
- **Sense без translations:** Translations = [] — допустимо. Переводы можно добавить позже.
- **PartOfSpeech = nil:** Часть речи не указана — допустимо.
- **Entry soft-deleted:** checkEntryOwnership вернёт ErrNotFound (GetByID фильтрует deleted_at IS NULL).
- **Position:** Repo автоматически назначает MAX(position)+1. Сервис не передаёт position при создании.

---

### 4.2. UpdateSense (C2)

**Сценарий:** Пользователь изменяет definition/pos/cefr у sense. Partial customization — ref_sense_id не трогается, origin link сохраняется.

**Метод:** `UpdateSense(ctx context.Context, input UpdateSenseInput) → (*domain.Sense, error)`

**UpdateSenseInput:**
- SenseID (uuid.UUID)
- Definition (*string) — новое значение (nil = не менять, не "очистить")
- PartOfSpeech (*domain.PartOfSpeech) — nil = не менять
- CEFRLevel (*string) — nil = не менять

**Семантика nil vs explicit:** Для UpdateSense используется **partial update** паттерн. Nil поле означает "не менять". Это отличается от domain COALESCE-семантики (где NULL = "наследовать из ref"). На уровне repo: Update устанавливает только non-nil поля.

Если пользователь хочет **вернуть наследование** из каталога (сбросить свою кастомизацию) — это отдельная операция ResetSenseField (не реализуется на MVP, т.к. UI для этого нет).

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. oldSense, entry, err = s.checkSenseOwnership(ctx, userID, input.SenseID)
   └─ ErrNotFound → return ErrNotFound

4. txManager.RunInTx(ctx, func(ctx) error {

   4a. sense, err = senseRepo.Update(ctx, input.SenseID, input.Definition, input.PartOfSpeech, input.CEFRLevel)
       // ref_sense_id НЕ трогается — это инвариант repo

   4b. auditRepo.Log(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &input.SenseID,
           Action:     domain.ActionUpdate,
           Changes:    buildSenseChanges(oldSense, input),
       })

   return nil
   })

5. return sense, nil
```

**Corner cases:**
- **Partial customization:** Пользователь меняет definition, оставляя pos из каталога. Repo обновляет только definition, pos остаётся NULL → COALESCE подхватит ref-значение.
- **Все поля nil:** Формально ничего не меняется, но UPDATE + audit выполняются. Допустимо (проще, чем проверять "есть ли изменения").
- **Sense без ref:** Sense создан custom (ref_sense_id = NULL). Update работает аналогично — просто обновляет user-поля.

---

### 4.3. DeleteSense (C3)

**Сценарий:** Пользователь удаляет sense. CASCADE удаляет все дочерние translations и examples.

**Метод:** `DeleteSense(ctx context.Context, senseID uuid.UUID) → error`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)

2. sense, entry, err = s.checkSenseOwnership(ctx, userID, senseID)
   └─ ErrNotFound → return ErrNotFound

3. txManager.RunInTx(ctx, func(ctx) error {

   3a. senseRepo.Delete(ctx, senseID)
       // CASCADE: translations, examples удаляются автоматически

   3b. auditRepo.Log(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &senseID,
           Action:     domain.ActionDelete,
           Changes: map[string]any{
               "entry_id":   {"old": sense.EntryID},
               "definition": {"old": sense.Definition},
           },
       })

   return nil
   })

4. return nil
```

**Corner cases:**
- **Удаление последнего sense:** Допустимо. Entry может временно остаться без senses. Пользователь может добавить новые позже.
- **CASCADE:** Translations и examples удаляются автоматически FK ON DELETE CASCADE. Сервис не удаляет их явно.
- **Position gap:** Удаление sense оставляет gap в позициях остальных senses. Это ожидаемое поведение — gap не влияет на ORDER BY.

---

### 4.4. ReorderSenses (C10 — senses)

**Сценарий:** Пользователь меняет порядок значений перетаскиванием.

**Метод:** `ReorderSenses(ctx context.Context, input ReorderSensesInput) → error`

**ReorderSensesInput:**
- EntryID (uuid.UUID)
- Items ([]ReorderItem) — [{ID, Position}]

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. entry, err = s.checkEntryOwnership(ctx, userID, input.EntryID)
   └─ ErrNotFound → return ErrNotFound

4. Валидация items:
   existingSenses, err = senseRepo.GetByEntryID(ctx, input.EntryID)
   existingIDs = set(existingSenses.IDs)

   Для каждого item в input.Items:
     if item.ID not in existingIDs → return ValidationError("items", "sense does not belong to this entry: " + item.ID)

5. senseRepo.Reorder(ctx, input.Items)
   // Repo обновляет позиции батчем в одной транзакции

6. return nil
```

**Corner cases:**
- **Partial list:** Клиент отправляет не все senses. Repo обновляет только переданные. Остальные сохраняют свои позиции. Это ожидаемое поведение.
- **Duplicate positions в input:** Допустимо — ORDER BY по (position, id) разрешит неоднозначность.
- **ID из другого entry:** Валидация на шаге 4 отклонит.
- **Пустой Items:** Валидация: len(Items) > 0.

---

### 4.5. AddTranslation (C4)

**Сценарий:** Пользователь добавляет свой перевод к значению.

**Метод:** `AddTranslation(ctx context.Context, input AddTranslationInput) → (*domain.Translation, error)`

**AddTranslationInput:**
- SenseID (uuid.UUID)
- Text (string)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. _, entry, err = s.checkSenseOwnership(ctx, userID, input.SenseID)
   └─ ErrNotFound → return ErrNotFound

4. count, err = translationRepo.CountBySense(ctx, input.SenseID)
   └─ count >= MaxTranslationsPerSense → return ValidationError("translations", "limit reached (20)")

5. txManager.RunInTx(ctx, func(ctx) error {

   5a. translation, err = translationRepo.CreateCustom(ctx, input.SenseID, input.Text, "user")

   5b. auditRepo.Log(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &input.SenseID,
           Action:     domain.ActionUpdate,
           Changes: map[string]any{
               "translation_added": {"new": input.Text},
           },
       })

   return nil
   })

6. return translation, nil
```

**Audit note:** Translations аудитируются как UPDATE на parent Sense (решение из service_layer_spec — чтобы не раздувать audit log). EntityType = SENSE, EntityID = senseID.

---

### 4.6. UpdateTranslation (C5)

**Сценарий:** Пользователь изменяет текст перевода. ref_translation_id сохраняется (origin link).

**Метод:** `UpdateTranslation(ctx context.Context, input UpdateTranslationInput) → (*domain.Translation, error)`

**UpdateTranslationInput:**
- TranslationID (uuid.UUID)
- Text (string)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. oldTranslation, entry, err = s.checkTranslationOwnership(ctx, userID, input.TranslationID)
   └─ ErrNotFound → return ErrNotFound

4. txManager.RunInTx(ctx, func(ctx) error {

   4a. translation, err = translationRepo.Update(ctx, input.TranslationID, input.Text)
       // ref_translation_id НЕ трогается

   4b. auditRepo.Log(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &oldTranslation.SenseID,
           Action:     domain.ActionUpdate,
           Changes: map[string]any{
               "translation_text": {"old": oldTranslation.Text, "new": input.Text},
           },
       })

   return nil
   })

5. return translation, nil
```

**Corner cases:**
- **Текст не изменился:** UPDATE + audit всё равно выполняются. Проще и предсказуемее.
- **Translation из каталога:** ref_translation_id сохраняется. User-поле text перезаписывается. COALESCE теперь вернёт user-значение вместо ref-значения.

---

### 4.7. DeleteTranslation (C6)

**Метод:** `DeleteTranslation(ctx context.Context, translationID uuid.UUID) → error`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)

2. translation, entry, err = s.checkTranslationOwnership(ctx, userID, translationID)
   └─ ErrNotFound → return ErrNotFound

3. txManager.RunInTx(ctx, func(ctx) error {
   3a. translationRepo.Delete(ctx, translationID)
   3b. auditRepo.Log(ctx, ...) // EntitySense, ActionUpdate
   return nil
   })

4. return nil
```

**Corner cases:**
- **Удаление последнего перевода:** Допустимо. Sense может существовать без переводов.

---

### 4.8. ReorderTranslations (C10 — translations)

**Метод:** `ReorderTranslations(ctx context.Context, input ReorderTranslationsInput) → error`

**ReorderTranslationsInput:**
- SenseID (uuid.UUID)
- Items ([]ReorderItem)

**Flow:** Аналогичен ReorderSenses, но:
- Ownership check через checkSenseOwnership
- Валидация items через translationRepo.GetBySenseID
- Repo call: translationRepo.Reorder

---

### 4.9. AddExample (C7)

**Сценарий:** Пользователь добавляет свой пример (sentence + translation).

**Метод:** `AddExample(ctx context.Context, input AddExampleInput) → (*domain.Example, error)`

**AddExampleInput:**
- SenseID (uuid.UUID)
- Sentence (string)
- Translation (*string)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. _, entry, err = s.checkSenseOwnership(ctx, userID, input.SenseID)
   └─ ErrNotFound → return ErrNotFound

4. count, err = exampleRepo.CountBySense(ctx, input.SenseID)
   └─ count >= MaxExamplesPerSense → return ValidationError("examples", "limit reached (50)")

5. txManager.RunInTx(ctx, func(ctx) error {
   5a. example, err = exampleRepo.CreateCustom(ctx, input.SenseID, input.Sentence, input.Translation, "user")
   5b. auditRepo.Log(ctx, ...) // EntitySense, ActionUpdate
   return nil
   })

6. return example, nil
```

---

### 4.10. UpdateExample (C8)

**Метод:** `UpdateExample(ctx context.Context, input UpdateExampleInput) → (*domain.Example, error)`

**UpdateExampleInput:**
- ExampleID (uuid.UUID)
- Sentence (string)
- Translation (*string) — nil = убрать перевод примера

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. oldExample, entry, err = s.checkExampleOwnership(ctx, userID, input.ExampleID)
   └─ ErrNotFound → return ErrNotFound

4. txManager.RunInTx(ctx, func(ctx) error {
   4a. example, err = exampleRepo.Update(ctx, input.ExampleID, input.Sentence, input.Translation)
   4b. auditRepo.Log(ctx, ...) // EntitySense, ActionUpdate
   return nil
   })

5. return example, nil
```

**Corner cases:**
- **Translation = nil:** Перевод примера убирается (set translation = NULL в БД). Это **не** COALESCE-семантика: для examples.translation NULL означает "нет перевода", а не "наследовать". COALESCE здесь работает только если есть ref_example_id и ref_example имеет translation.

---

### 4.11. DeleteExample (C9)

**Метод:** `DeleteExample(ctx context.Context, exampleID uuid.UUID) → error`

Flow аналогичен DeleteTranslation: ownership check → delete → audit.

---

### 4.12. ReorderExamples (C10 — examples)

**Метод:** `ReorderExamples(ctx context.Context, input ReorderExamplesInput) → error`

Flow аналогичен ReorderSenses/ReorderTranslations, но через exampleRepo.

---

### 4.13. AddUserImage (C11)

**Сценарий:** Пользователь добавляет своё изображение к слову.

**Метод:** `AddUserImage(ctx context.Context, input AddUserImageInput) → (*domain.UserImage, error)`

**AddUserImageInput:**
- EntryID (uuid.UUID)
- URL (string)
- Caption (*string)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. entry, err = s.checkEntryOwnership(ctx, userID, input.EntryID)
   └─ ErrNotFound → return ErrNotFound

4. image, err = imageRepo.CreateUser(ctx, input.EntryID, input.URL, input.Caption)

5. return image, nil
```

**Нет транзакции** (одна операция). **Нет audit** (user images — lightweight контент, не аудитируем).

**Corner cases:**
- **Лимит на images:** На MVP нет лимита на user images per entry. При необходимости добавить в будущем.
- **URL validation:** Проверяется в Validate() — должен быть HTTP(S) URL. Содержимое по URL не проверяется.
- **Кто хостит изображения:** Вне scope этого сервиса. URL приходит от клиента (клиент загружает в S3/CDN и передаёт URL). Обработка загрузки — отдельный компонент (не ContentService).

---

### 4.14. DeleteUserImage (C12)

**Метод:** `DeleteUserImage(ctx context.Context, imageID uuid.UUID) → error`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)

2. image, entry, err = s.checkUserImageOwnership(ctx, userID, imageID)
   └─ ErrNotFound → return ErrNotFound

3. imageRepo.DeleteUser(ctx, imageID)

4. return nil
```

**Нет audit.** Нет транзакции.

---

## 5. Input Validation

### 5.1. AddSenseInput

```
Validate():
    if EntryID == uuid.Nil               → ("entry_id", "required")
    if Definition != nil && len > 2000    → ("definition", "too long (max 2000)")
    if CEFRLevel != nil && len > 10       → ("cefr_level", "too long")
    if len(Translations) > 20            → ("translations", "too many (max 20)")
    for each tr in Translations:
      if tr == "" after trim              → ("translations[i]", "required")
      if len(tr) > 500                   → ("translations[i]", "too long (max 500)")
```

### 5.2. UpdateSenseInput

```
Validate():
    if SenseID == uuid.Nil                → ("sense_id", "required")
    if Definition != nil && len > 2000    → ("definition", "too long")
    if CEFRLevel != nil && len > 10       → ("cefr_level", "too long")
```

### 5.3. ReorderSensesInput / ReorderTranslationsInput / ReorderExamplesInput

```
Validate():
    if ParentID == uuid.Nil               → ("entry_id"/"sense_id", "required")
    if len(Items) == 0                    → ("items", "required")
    if len(Items) > 50                    → ("items", "too many")
    for each item:
      if item.ID == uuid.Nil              → ("items[i].id", "required")
      if item.Position < 0               → ("items[i].position", "must be >= 0")
```

### 5.4. AddTranslationInput

```
Validate():
    if SenseID == uuid.Nil                → ("sense_id", "required")
    if Text == "" after trim              → ("text", "required")
    if len(Text) > 500                    → ("text", "too long (max 500)")
```

### 5.5. UpdateTranslationInput

```
Validate():
    if TranslationID == uuid.Nil          → ("translation_id", "required")
    if Text == "" after trim              → ("text", "required")
    if len(Text) > 500                    → ("text", "too long")
```

### 5.6. AddExampleInput

```
Validate():
    if SenseID == uuid.Nil                → ("sense_id", "required")
    if Sentence == "" after trim          → ("sentence", "required")
    if len(Sentence) > 2000              → ("sentence", "too long (max 2000)")
    if Translation != nil && len > 2000   → ("translation", "too long")
```

### 5.7. UpdateExampleInput

```
Validate():
    if ExampleID == uuid.Nil              → ("example_id", "required")
    if Sentence == "" after trim          → ("sentence", "required")
    if len(Sentence) > 2000              → ("sentence", "too long")
    if Translation != nil && len > 2000   → ("translation", "too long")
```

### 5.8. AddUserImageInput

```
Validate():
    if EntryID == uuid.Nil                → ("entry_id", "required")
    if URL == "" after trim               → ("url", "required")
    if !isValidHTTPURL(URL)               → ("url", "must be a valid HTTP(S) URL")
    if len(URL) > 2000                    → ("url", "too long")
    if Caption != nil && len > 500        → ("caption", "too long (max 500)")
```

---

## 6. Error Scenarios — полная таблица

| Операция | Условие | Ошибка |
|----------|---------|--------|
| Все мутации | Нет userID в ctx | ErrUnauthorized |
| Все мутации | Невалидный input | ValidationError |
| AddSense | Entry не найден / deleted / чужой | ErrNotFound |
| AddSense | Лимит senses (20) | ValidationError("senses", "limit reached") |
| UpdateSense | Sense не найден | ErrNotFound |
| UpdateSense | Entry (через sense) чужой / deleted | ErrNotFound |
| DeleteSense | Sense не найден или чужой | ErrNotFound |
| ReorderSenses | Entry не найден / чужой | ErrNotFound |
| ReorderSenses | Sense ID не принадлежит entry | ValidationError("items") |
| AddTranslation | Sense не найден или чужой | ErrNotFound |
| AddTranslation | Лимит translations (20) | ValidationError("translations", "limit reached") |
| UpdateTranslation | Translation не найден или чужой chain | ErrNotFound |
| DeleteTranslation | Translation не найден или чужой | ErrNotFound |
| ReorderTranslations | Sense не найден или чужой | ErrNotFound |
| ReorderTranslations | Translation ID не принадлежит sense | ValidationError("items") |
| AddExample | Sense не найден или чужой | ErrNotFound |
| AddExample | Лимит examples (50) | ValidationError("examples", "limit reached") |
| UpdateExample | Example не найден или чужой chain | ErrNotFound |
| DeleteExample | Example не найден или чужой | ErrNotFound |
| ReorderExamples | аналогично ReorderTranslations | |
| AddUserImage | Entry не найден / deleted / чужой | ErrNotFound |
| DeleteUserImage | Image не найден или чужой chain | ErrNotFound |

**Паттерн ownership:** Если элемент существует, но принадлежит чужому entry или entry soft-deleted — всегда ErrNotFound (не ErrForbidden). Не раскрываем существование чужих данных.

---

## 7. Тесты

### 7.1. Моки

Все зависимости мокаются: entryRepo, senseRepo, translationRepo, exampleRepo, imageRepo, auditRepo, txManager.

### 7.2. Тест-кейсы

**Ownership helpers (тестируются через public методы):**

| # | Тест | Assert |
|---|------|--------|
| OW1 | Entry принадлежит пользователю | Operation succeeds |
| OW2 | Entry не найден | ErrNotFound |
| OW3 | Entry soft-deleted | ErrNotFound |
| OW4 | Sense → entry чужой | ErrNotFound |
| OW5 | Translation → sense → entry чужой | ErrNotFound |

**AddSense:**

| # | Тест | Assert |
|---|------|--------|
| AS1 | Happy path — с definition и translations | Sense создан. Translations создаются. Audit logged. |
| AS2 | Без definition | Sense создан с definition=nil |
| AS3 | Без translations | Sense создан. translationRepo.CreateCustom NOT called |
| AS4 | Лимит senses | ValidationError "limit reached" |
| AS5 | Entry не найден | ErrNotFound |
| AS6 | Entry deleted | ErrNotFound |
| AS7 | Невалидный input | ValidationError |
| AS8 | Нет userID | ErrUnauthorized |
| AS9 | source_slug = "user" | CreateCustom called with sourceSlug="user" |

**UpdateSense:**

| # | Тест | Assert |
|---|------|--------|
| US1 | Изменить definition | senseRepo.Update called. Audit logged with old/new. |
| US2 | Изменить pos | senseRepo.Update called with pos |
| US3 | Partial: только definition, pos не трогать | Update called. ref_sense_id NOT changed. |
| US4 | Все поля nil | Update called (no-op but allowed) |
| US5 | Sense не найден | ErrNotFound |
| US6 | Sense чужого entry | ErrNotFound |

**DeleteSense:**

| # | Тест | Assert |
|---|------|--------|
| DS1 | Happy path | senseRepo.Delete called. Audit logged. |
| DS2 | Последний sense entry | Delete succeeds (entry без senses допустим) |
| DS3 | Sense не найден | ErrNotFound |
| DS4 | Audit содержит definition | Changes include old definition |

**ReorderSenses:**

| # | Тест | Assert |
|---|------|--------|
| RS1 | Happy path | senseRepo.Reorder called with items |
| RS2 | Entry не найден | ErrNotFound |
| RS3 | Sense ID из другого entry | ValidationError "does not belong" |
| RS4 | Partial list (не все senses) | Reorder called only with provided items |
| RS5 | Пустой Items | ValidationError |

**AddTranslation:**

| # | Тест | Assert |
|---|------|--------|
| AT1 | Happy path | Translation создан. Audit на parent sense. |
| AT2 | Лимит translations | ValidationError "limit reached" |
| AT3 | Sense не найден | ErrNotFound |
| AT4 | Sense чужого entry | ErrNotFound |

**UpdateTranslation:**

| # | Тест | Assert |
|---|------|--------|
| UT1 | Happy path | translationRepo.Update called. ref_translation_id не трогается. |
| UT2 | Translation не найден | ErrNotFound |
| UT3 | Audit содержит old/new text | Changes: {"translation_text": {"old": ..., "new": ...}} |

**DeleteTranslation:**

| # | Тест | Assert |
|---|------|--------|
| DT1 | Happy path | Deleted. Audit logged. |
| DT2 | Последний translation sense | Delete succeeds |
| DT3 | Translation не найден | ErrNotFound |

**ReorderTranslations:**

| # | Тест | Assert |
|---|------|--------|
| RT1 | Happy path | translationRepo.Reorder called |
| RT2 | Translation ID из другого sense | ValidationError |

**AddExample:**

| # | Тест | Assert |
|---|------|--------|
| AE1 | Happy path — sentence + translation | Example создан. |
| AE2 | Без translation | Example создан с translation=nil |
| AE3 | Лимит examples (50) | ValidationError |
| AE4 | Sense не найден | ErrNotFound |

**UpdateExample:**

| # | Тест | Assert |
|---|------|--------|
| UE1 | Изменить sentence | exampleRepo.Update called |
| UE2 | Translation = nil (убрать перевод) | Update called with translation=nil |
| UE3 | Example не найден | ErrNotFound |

**DeleteExample:**

| # | Тест | Assert |
|---|------|--------|
| DEX1 | Happy path | Deleted. Audit logged. |
| DEX2 | Example не найден | ErrNotFound |

**ReorderExamples:**

| # | Тест | Assert |
|---|------|--------|
| REX1 | Happy path | exampleRepo.Reorder called |
| REX2 | Example ID из другого sense | ValidationError |

**AddUserImage:**

| # | Тест | Assert |
|---|------|--------|
| AI1 | Happy path | Image created. No audit. |
| AI2 | Entry не найден | ErrNotFound |
| AI3 | Невалидный URL | ValidationError |
| AI4 | С caption | Image.Caption set |
| AI5 | Без caption | Image.Caption = nil |

**DeleteUserImage:**

| # | Тест | Assert |
|---|------|--------|
| DI1 | Happy path | Deleted. No audit. |
| DI2 | Image не найден | ErrNotFound |
| DI3 | Image чужого entry | ErrNotFound |

---

## 8. Файловая структура

```
internal/service/content/
├── service.go           # Service struct, конструктор, приватные интерфейсы,
│                        #   ownership helpers,
│                        #   AddSense, UpdateSense, DeleteSense, ReorderSenses,
│                        #   AddTranslation, UpdateTranslation, DeleteTranslation, ReorderTranslations,
│                        #   AddExample, UpdateExample, DeleteExample, ReorderExamples,
│                        #   AddUserImage, DeleteUserImage
├── input.go             # Все input-структуры + Validate()
└── service_test.go      # Все тесты из секции 7 (~50 тестов)
```

---

## 9. Взаимодействие с другими компонентами

### 9.1. DictionaryService (нет зависимости)

DictionaryService создаёт начальный контент при добавлении слова. ContentService управляет дальнейшим редактированием. Оба работают с repo напрямую, не вызывают друг друга.

### 9.2. Transport Layer (GraphQL)

Resolver определяет узкий интерфейс:

```
type contentService interface {
    AddSense(ctx, input) → (*Sense, error)
    UpdateSense(ctx, input) → (*Sense, error)
    DeleteSense(ctx, senseID) → error
    ReorderSenses(ctx, input) → error

    AddTranslation(ctx, input) → (*Translation, error)
    UpdateTranslation(ctx, input) → (*Translation, error)
    DeleteTranslation(ctx, translationID) → error
    ReorderTranslations(ctx, input) → error

    AddExample(ctx, input) → (*Example, error)
    UpdateExample(ctx, input) → (*Example, error)
    DeleteExample(ctx, exampleID) → error
    ReorderExamples(ctx, input) → error

    AddUserImage(ctx, input) → (*UserImage, error)
    DeleteUserImage(ctx, imageID) → error
}
```

### 9.3. DataLoaders (нет взаимодействия)

Чтение дочернего контента (senses by entry, translations by sense, и т.д.) осуществляется через DataLoaders, которые вызывают repo напрямую. ContentService не участвует в чтении batch-данных.
