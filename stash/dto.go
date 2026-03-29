package stash

type Message struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

type Page struct {
	Limit  uint
	Offset uint
	Total  uint
}
