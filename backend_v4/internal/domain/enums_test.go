package domain

import "testing"

func TestLearningStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status LearningStatus
		want   bool
	}{
		{LearningStatusNew, true},
		{LearningStatusLearning, true},
		{LearningStatusReview, true},
		{LearningStatusMastered, true},
		{LearningStatus("INVALID"), false},
		{LearningStatus(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("LearningStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestLearningStatus_String(t *testing.T) {
	t.Parallel()
	if got := LearningStatusNew.String(); got != "NEW" {
		t.Errorf("got %q, want NEW", got)
	}
}

func TestReviewGrade_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		grade ReviewGrade
		want  bool
	}{
		{ReviewGradeAgain, true},
		{ReviewGradeHard, true},
		{ReviewGradeGood, true},
		{ReviewGradeEasy, true},
		{ReviewGrade("INVALID"), false},
		{ReviewGrade(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.grade), func(t *testing.T) {
			t.Parallel()
			if got := tt.grade.IsValid(); got != tt.want {
				t.Errorf("ReviewGrade(%q).IsValid() = %v, want %v", tt.grade, got, tt.want)
			}
		})
	}
}

func TestReviewGrade_String(t *testing.T) {
	t.Parallel()
	if got := ReviewGradeAgain.String(); got != "AGAIN" {
		t.Errorf("got %q, want AGAIN", got)
	}
}

func TestPartOfSpeech_IsValid(t *testing.T) {
	t.Parallel()

	valid := []PartOfSpeech{
		PartOfSpeechNoun, PartOfSpeechVerb, PartOfSpeechAdjective, PartOfSpeechAdverb,
		PartOfSpeechPronoun, PartOfSpeechPreposition, PartOfSpeechConjunction,
		PartOfSpeechInterjection, PartOfSpeechPhrase, PartOfSpeechIdiom, PartOfSpeechOther,
	}
	for _, p := range valid {
		if !p.IsValid() {
			t.Errorf("PartOfSpeech(%q).IsValid() = false, want true", p)
		}
	}
	if PartOfSpeech("UNKNOWN").IsValid() {
		t.Error("PartOfSpeech(UNKNOWN).IsValid() = true, want false")
	}
}

func TestPartOfSpeech_String(t *testing.T) {
	t.Parallel()
	if got := PartOfSpeechNoun.String(); got != "NOUN" {
		t.Errorf("got %q, want NOUN", got)
	}
}

func TestEntityType_IsValid(t *testing.T) {
	t.Parallel()

	valid := []EntityType{
		EntityTypeEntry, EntityTypeSense, EntityTypeExample, EntityTypeImage,
		EntityTypePronunciation, EntityTypeCard, EntityTypeTopic, EntityTypeUser,
	}
	for _, e := range valid {
		if !e.IsValid() {
			t.Errorf("EntityType(%q).IsValid() = false, want true", e)
		}
	}
	if EntityType("BOGUS").IsValid() {
		t.Error("EntityType(BOGUS).IsValid() = true, want false")
	}
}

func TestEntityType_String(t *testing.T) {
	t.Parallel()
	if got := EntityTypeTopic.String(); got != "TOPIC" {
		t.Errorf("got %q, want TOPIC", got)
	}
}

func TestAuditAction_IsValid(t *testing.T) {
	t.Parallel()

	valid := []AuditAction{AuditActionCreate, AuditActionUpdate, AuditActionDelete}
	for _, a := range valid {
		if !a.IsValid() {
			t.Errorf("AuditAction(%q).IsValid() = false, want true", a)
		}
	}
	if AuditAction("NOPE").IsValid() {
		t.Error("AuditAction(NOPE).IsValid() = true, want false")
	}
}

func TestAuditAction_String(t *testing.T) {
	t.Parallel()
	if got := AuditActionCreate.String(); got != "CREATE" {
		t.Errorf("got %q, want CREATE", got)
	}
}

func TestAuthMethodType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method AuthMethodType
		want   bool
	}{
		{AuthMethodPassword, true},
		{AuthMethodGoogle, true},
		{AuthMethodApple, true},
		{AuthMethodType("facebook"), false},
		{AuthMethodType(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			t.Parallel()
			if got := tt.method.IsValid(); got != tt.want {
				t.Errorf("AuthMethodType(%q).IsValid() = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestAuthMethodType_IsOAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method AuthMethodType
		want   bool
	}{
		{AuthMethodGoogle, true},
		{AuthMethodApple, true},
		{AuthMethodPassword, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			t.Parallel()
			if got := tt.method.IsOAuth(); got != tt.want {
				t.Errorf("AuthMethodType(%q).IsOAuth() = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestSessionStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status SessionStatus
		want   bool
	}{
		{SessionStatusActive, true},
		{SessionStatusFinished, true},
		{SessionStatusAbandoned, true},
		{SessionStatus("INVALID"), false},
		{SessionStatus(""), false},
		{SessionStatus("PENDING"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("SessionStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestSessionStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status SessionStatus
		want   string
	}{
		{SessionStatusActive, "ACTIVE"},
		{SessionStatusFinished, "FINISHED"},
		{SessionStatusAbandoned, "ABANDONED"},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			if got := tt.status.String(); got != tt.want {
				t.Errorf("SessionStatus(%q).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
