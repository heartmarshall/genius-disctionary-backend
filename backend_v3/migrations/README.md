# Миграции базы данных

Этот каталог содержит миграции базы данных, управляемые с помощью [goose](https://github.com/pressly/goose).

## Структура миграций

Миграции следуют формату `YYYYMMDDHHMMSS_description.sql` и используют директивы goose:
- `-- +goose Up` - код для применения миграции
- `-- +goose Down` - код для отката миграции

Оба блока находятся в одном файле.

## Использование

### Применить все миграции
```bash
make migrate-up DB_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
```

### Откатить последнюю миграцию
```bash
make migrate-down DB_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
```

### Проверить статус миграций
```bash
make migrate-status DB_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
```

### Создать новую миграцию
```bash
make migrate-create NAME=add_new_feature
```
## Примечания

- Все миграции должны быть обратимыми
- При создании новых миграций используйте команду `make migrate-create`
- Перед применением миграций убедитесь, что база данных существует

