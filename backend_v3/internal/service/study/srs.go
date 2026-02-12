package study

import (
	"math"
	"time"

	"github.com/heartmarshall/my-english/internal/model"
)

// SRSResult — результат вычисления алгоритма.
type SRSResult struct {
	Status       model.LearningStatus
	NextReviewAt time.Time
	IntervalDays int
	EaseFactor   float64
}

// calculateSRS рассчитывает новые параметры карточки на основе оценки.
// now — текущее время (передаем явно для тестируемости).
func calculateSRS(card *model.Card, grade model.ReviewGrade, now time.Time) SRSResult {
	// Базовые константы
	const (
		minEaseFactor = 1.3
		bonusEasy     = 1.3 // Бонус к интервалу для легких ответов
	)

	// Копируем текущие значения
	nextInterval := card.IntervalDays
	nextEase := card.EaseFactor
	nextStatus := card.Status

	// Логика переходов состояний
	switch grade {
	case model.GradeAgain:
		// Сброс прогресса
		nextInterval = 0 // или 1 день, зависит от жесткости
		nextEase = math.Max(minEaseFactor, nextEase-0.2)
		nextStatus = model.StatusLearning // Возвращаем в обучение

	case model.GradeHard:
		// Интервал растет медленно (x1.2)
		if nextInterval == 0 {
			nextInterval = 1
		} else {
			nextInterval = int(float64(nextInterval) * 1.2)
		}
		nextEase = math.Max(minEaseFactor, nextEase-0.15)
		nextStatus = model.StatusReview

	case model.GradeGood:
		// Стандартный SM-2: Interval * EF
		if nextInterval == 0 {
			nextInterval = 1
		} else if nextInterval == 1 {
			nextInterval = 3 // Второй шаг часто фиксирован
		} else {
			nextInterval = int(float64(nextInterval) * nextEase)
		}
		// Ease не меняется или немного растет? В классике SM-2 он меняется по формуле.
		// Упростим: для Good EF остается прежним.
		nextStatus = model.StatusReview

	case model.GradeEasy:
		// Быстрый рост: Interval * EF * Bonus
		if nextInterval == 0 {
			nextInterval = 4
		} else {
			nextInterval = int(float64(nextInterval) * nextEase * bonusEasy)
		}
		nextEase += 0.15
		nextStatus = model.StatusReview
	}

	// Если статус был NEW, он всегда меняется на LEARNING (при Again) или REVIEW (остальные)
	if card.Status == model.StatusNew {
		if grade == model.GradeAgain {
			nextStatus = model.StatusLearning
		} else {
			nextStatus = model.StatusReview
		}
	}

	// Рассчитываем дату следующего повторения
	// Для "Again" (Interval=0) ставим, например, +10 минут или +1 час.
	// Но так как у нас в базе IntervalDays (int), упростим для MVP:
	// Interval 0 -> NextReviewAt = Now (в очередь сразу) или Now + 5 min.
	var nextReviewAt time.Time
	if nextInterval == 0 {
		// Повторить сегодня (через пару минут)
		nextReviewAt = now.Add(10 * time.Minute)
	} else {
		// Повторить через N дней
		nextReviewAt = now.AddDate(0, 0, nextInterval)
	}

	return SRSResult{
		Status:       nextStatus,
		NextReviewAt: nextReviewAt,
		IntervalDays: nextInterval,
		EaseFactor:   nextEase,
	}
}
