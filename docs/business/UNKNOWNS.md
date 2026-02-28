# Unknowns and Open Questions

These are things the code doesn't make clear, or where business intent is ambiguous. Each needs verification from someone who knows the domain.

## Ambiguous Business Rules

| # | Question | What the code does | Why it's unclear |
|---|---|---|---|
| 1 | What happens when a user exceeds the entry limit mid-import? | The system pre-checks `current_count + import_size > limit` before starting, but per-chunk processing could theoretically allow slight over-limit due to concurrent imports | Is concurrent import a real scenario? Should there be per-entry counting within the import transaction? |
| 2 | Can a user have entries without any senses? | Custom entries don't require senses — only flashcard creation requires at least one sense | Is a senseless entry intentional (placeholder) or a data quality issue? |
| 3 | What is the intended relationship between Inbox and Dictionary? | Inbox items are completely disconnected from entries — just free text | Should processing an inbox item auto-search the catalog? Should there be a "convert to entry" action? |
| 4 | Is ReviewsPerDay purely cosmetic? | The code comments say "not enforced in queue" and "informational goal shown in dashboard UI" | If it's never enforced, why is it configurable? Is it shown in the frontend? |
| 5 | What translation provider is used? | Abstracted behind `internal/adapter/provider/translate/` interface | Google Translate? DeepL? A custom solution? |
| 6 | Should enrichment failures be retried automatically? | Currently requires manual admin intervention to retry failed items | Is auto-retry intentional? What's the expected failure rate? |
| 7 | What happens to cards when their entry is soft-deleted? | The code soft-deletes the entry, but the card and review logs appear to remain | Should cards be suspended? Should they still appear in the study queue? |

## Missing Business Logic (possibly)

| # | What seems absent | Why it might matter |
|---|---|---|
| 1 | No per-user entry deduplication across normalized variants (e.g., "run" vs "Run" is handled, but "colour" vs "color" is not) | Users might accidentally add spelling variants as separate entries |
| 2 | No "forgot password" / password reset flow | Users who registered with password and forget it have no recovery path (unless they also linked OAuth) |
| 3 | No email verification on registration | Fake/typo emails can be used to register — could cause account linking issues if a real owner later registers via OAuth |
| 4 | No session timeout for study sessions | A session started and never finished stays ACTIVE indefinitely — there's no auto-abandon or auto-finish |
| 5 | No notification system (email, push) | No transactional emails (welcome, password reset) or study reminders |
| 6 | No user deletion / account removal | No GDPR-style "delete my account" functionality visible in the code |
| 7 | No export for user data | Export exists for entries, but no full data export (study history, settings, etc.) |
| 8 | No content moderation for custom entries | Users can write anything in custom definitions — no filtering or flagging |

## Assumptions Made in This Document

| # | Assumption | Basis |
|---|---|---|
| 1 | This is a mobile-first application | GraphQL API (common for mobile), Apple Sign-In support, timezone-aware study tracking |
| 2 | Target audience is non-native English speakers | Translations feature, CEFR levels (European language framework), app name "MyEnglish" |
| 3 | Single language pair: English → user's native language | No language selector in user settings; translations don't specify target language |
| 4 | The system operates with a single currency / is free to use | No payment, subscription, or premium tier logic anywhere in the code |
| 5 | Audit logs are for developer debugging, not user-facing | Audit records are created but no API to query them exists for regular users |
| 6 | "Core lexicon" flag on reference entries indicates high-priority words for learners | `IsCoreLexicon` boolean exists on RefEntry but its usage isn't clear from service logic |
| 7 | FSRS-5 model weights are intentionally hardcoded (not per-user) | The 19 weights use global defaults — no mechanism for per-user weight optimization exists |
