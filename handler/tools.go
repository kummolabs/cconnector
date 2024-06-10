package handler

import "fmt"

func BadRequestResponseBody(message string) map[string]any {
	return map[string]any{
		"message": fmt.Sprintf("Request has invalid format: `%s`", message),
	}
}

func UnprocessableEntityResponseBody(message string) map[string]any {
	return map[string]any{
		"message": fmt.Sprintf("Entity cannot be processed because of: `%s`", message),
	}
}

func InternalServerErrorResponseBody() map[string]any {
	return map[string]any{
		"message": "Internal server error",
	}
}
