package server

import _ "embed"

var (
	//go:embed assets/verify_console.html
	verifyConsoleHTML string
)

func VerifyConsolePageHTML() string {
	return verifyConsoleHTML
}
