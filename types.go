package miservice

type Device struct {
	Name     string `json:"name"`
	Model    string `json:"model"`
	DeviceID string `json:"did"`
	Token    string `json:"token"`
}

func T2[A, B any](a A, b B) Tuple2[A, B] {
	return Tuple2[A, B]{A: a, B: b}
}

type Tuple2[A, B any] struct {
	A A
	B B
}

func T3[A, B, C any](a A, b B, c C) Tuple3[A, B, C] {
	return Tuple3[A, B, C]{A: a, B: b, C: c}
}

type Tuple3[A, B, C any] struct {
	A A
	B B
	C C
}

type Spec struct {
	Status  string `json:"status"`
	Model   string `json:"model"`
	Version int64  `json:"version"`
	Type    string `json:"type"`
	Ts      int64  `json:"ts"`
}
