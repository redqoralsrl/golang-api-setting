package user

import "go-template/internal/metrics"

var (
	UsersCreatedTotal = metrics.NewSimpleCounter(
		"api_users_created_total",
		"Total number of successfully created users.",
	)

	UsersLoginTotal = metrics.NewSimpleCounter(
		"api_users_login_total",
		"Total number of successful user logins.",
	)
)
