package topic

// CreateTopicInput данные для создания топика.
type CreateTopicInput struct {
	Name        string
	Description *string
}

// UpdateTopicInput данные для обновления топика.
type UpdateTopicInput struct {
	ID          string
	Name        *string
	Description *string
}

// DeleteTopicInput данные для удаления топика.
type DeleteTopicInput struct {
	ID string
}
