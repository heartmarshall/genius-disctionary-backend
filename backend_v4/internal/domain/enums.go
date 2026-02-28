package domain

// CardState represents the FSRS-5 learning state of a card.
type CardState string

const (
	CardStateNew        CardState = "NEW"
	CardStateLearning   CardState = "LEARNING"
	CardStateReview     CardState = "REVIEW"
	CardStateRelearning CardState = "RELEARNING"
)

func (s CardState) String() string { return string(s) }

func (s CardState) IsValid() bool {
	switch s {
	case CardStateNew, CardStateLearning, CardStateReview, CardStateRelearning:
		return true
	}
	return false
}

// ReviewGrade represents the user's self-assessed recall quality.
type ReviewGrade string

const (
	ReviewGradeAgain ReviewGrade = "AGAIN"
	ReviewGradeHard  ReviewGrade = "HARD"
	ReviewGradeGood  ReviewGrade = "GOOD"
	ReviewGradeEasy  ReviewGrade = "EASY"
)

func (g ReviewGrade) String() string { return string(g) }

func (g ReviewGrade) IsValid() bool {
	switch g {
	case ReviewGradeAgain, ReviewGradeHard, ReviewGradeGood, ReviewGradeEasy:
		return true
	}
	return false
}

// PartOfSpeech represents the grammatical category of a word.
type PartOfSpeech string

const (
	PartOfSpeechNoun         PartOfSpeech = "NOUN"
	PartOfSpeechVerb         PartOfSpeech = "VERB"
	PartOfSpeechAdjective    PartOfSpeech = "ADJECTIVE"
	PartOfSpeechAdverb       PartOfSpeech = "ADVERB"
	PartOfSpeechPronoun      PartOfSpeech = "PRONOUN"
	PartOfSpeechPreposition  PartOfSpeech = "PREPOSITION"
	PartOfSpeechConjunction  PartOfSpeech = "CONJUNCTION"
	PartOfSpeechInterjection PartOfSpeech = "INTERJECTION"
	PartOfSpeechPhrase       PartOfSpeech = "PHRASE"
	PartOfSpeechIdiom        PartOfSpeech = "IDIOM"
	PartOfSpeechOther        PartOfSpeech = "OTHER"
)

func (p PartOfSpeech) String() string { return string(p) }

func (p PartOfSpeech) IsValid() bool {
	switch p {
	case PartOfSpeechNoun, PartOfSpeechVerb, PartOfSpeechAdjective, PartOfSpeechAdverb,
		PartOfSpeechPronoun, PartOfSpeechPreposition, PartOfSpeechConjunction,
		PartOfSpeechInterjection, PartOfSpeechPhrase, PartOfSpeechIdiom, PartOfSpeechOther:
		return true
	}
	return false
}

// EntityType identifies the kind of domain entity (used in audit logs).
type EntityType string

const (
	EntityTypeEntry         EntityType = "ENTRY"
	EntityTypeSense         EntityType = "SENSE"
	EntityTypeExample       EntityType = "EXAMPLE"
	EntityTypeImage         EntityType = "IMAGE"
	EntityTypePronunciation EntityType = "PRONUNCIATION"
	EntityTypeCard          EntityType = "CARD"
	EntityTypeTopic         EntityType = "TOPIC"
	EntityTypeUser          EntityType = "USER"
)

func (e EntityType) String() string { return string(e) }

func (e EntityType) IsValid() bool {
	switch e {
	case EntityTypeEntry, EntityTypeSense, EntityTypeExample, EntityTypeImage,
		EntityTypePronunciation, EntityTypeCard, EntityTypeTopic, EntityTypeUser:
		return true
	}
	return false
}

// AuditAction represents the kind of mutation recorded in the audit log.
type AuditAction string

const (
	AuditActionCreate AuditAction = "CREATE"
	AuditActionUpdate AuditAction = "UPDATE"
	AuditActionDelete AuditAction = "DELETE"
)

func (a AuditAction) String() string { return string(a) }

func (a AuditAction) IsValid() bool {
	switch a {
	case AuditActionCreate, AuditActionUpdate, AuditActionDelete:
		return true
	}
	return false
}

// SessionStatus represents the state of a study session.
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "ACTIVE"
	SessionStatusFinished  SessionStatus = "FINISHED"
	SessionStatusAbandoned SessionStatus = "ABANDONED"
)

func (s SessionStatus) String() string { return string(s) }

func (s SessionStatus) IsValid() bool {
	switch s {
	case SessionStatusActive, SessionStatusFinished, SessionStatusAbandoned:
		return true
	}
	return false
}

// UserRole represents the authorization level of a user.
type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

func (r UserRole) String() string { return string(r) }

func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleUser, UserRoleAdmin:
		return true
	}
	return false
}

func (r UserRole) IsAdmin() bool {
	return r == UserRoleAdmin
}
