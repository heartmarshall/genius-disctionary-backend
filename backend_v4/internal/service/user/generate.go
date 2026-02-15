package user

//go:generate moq -out user_repo_mock_test.go -pkg user . userRepo
//go:generate moq -out settings_repo_mock_test.go -pkg user . settingsRepo
//go:generate moq -out audit_repo_mock_test.go -pkg user . auditRepo
//go:generate moq -out tx_manager_mock_test.go -pkg user . txManager
