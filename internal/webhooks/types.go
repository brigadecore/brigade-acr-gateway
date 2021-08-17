package webhooks

type Event struct {
	Action  string  `json:"action"`
	Target  Target  `json:"target"`
	Request Request `json:"request"`
}

type Target struct {
	Repository string `json:"repository"`
}

type Request struct {
	Host string `json:"host"`
}
