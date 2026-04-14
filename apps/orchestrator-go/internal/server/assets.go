package server

import _ "embed"

var (
	//go:embed assets/verify_console.html
	verifyConsoleHTML string

	//go:embed assets/dashboard.html
	dashboardHTML string
)

func VerifyConsolePageHTML() string {
	return verifyConsoleHTML
}

func DashboardPageHTML() string {
	return dashboardHTML
}
