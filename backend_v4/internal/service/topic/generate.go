package topic

//go:generate moq -out mocks_test.go -pkg topic . topicRepo entryRepo auditLogger txManager
