package chaos

func NormalResponse(code int, body any) Response {
	return Response{
		Status:  normalStatus,
		Message: normalMsgMap[code],
		Body:    body,
	}
}

const (
	Health = iota
	Ok
	TaskCreated
)

var normalMsgMap = map[int]string{
	Health:      "ok",
	Ok:          "ok",
	TaskCreated: "task created",
}
