package chaos

type status string

var (
	errorStatus  status = "error"
	normalStatus status = "normal"
)

type Response struct {
	Status  status `json:"status"`
	Message string `json:"message"`
	Body    any    `json:"body,omitempty"`
}
