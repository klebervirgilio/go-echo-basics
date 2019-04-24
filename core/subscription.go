package core

// EmailVerificationResponse represents the mail checker response.
type EmailVerificationResponse struct {
	Email      string  `json:"email"`
	Suggestion string  `json:"did_you_mean"`
	Valid      bool    `json:"format_valid"`
	Score      float64 `json:"score"`
}

// Subscription represents a mailist subscription.
type Subscription struct {
	EmailVerificationResponse `bson:"emailVerificationResponse"`
	Email                     string `bson:"email"`
	Name                      string `bson:"fullName"`
}

// Repository abstracts the application persistance layer.
type Repository interface {
	FindAll(selector map[string]interface{}) ([]Subscription, error)
	Remove(selector map[string]interface{}) error
	Upsert(subscription Subscription) error
}

type MailChecker interface {
	Validate(email string) (EmailVerificationResponse, error)
}
