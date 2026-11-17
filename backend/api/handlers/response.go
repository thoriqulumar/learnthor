package handlers

// errorResponse is the standard error envelope returned on all non-2xx responses.
// Defined here once and shared across all handlers in this package.
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
