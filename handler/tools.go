package handler

import "fmt"

func BadRequestResponseBody(message string) map[string]any {
	return map[string]any{
		"message": fmt.Sprintf("Request has invalid format: `%s`", message),
	}
}

func InternalServerErrorResponseBody() map[string]any {
	return map[string]any{
		"message": "Internal server error",
	}
}
