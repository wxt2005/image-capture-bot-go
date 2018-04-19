package model

type Media struct {
	FileName string
	URL      string
	File     *[]byte `json:"-"`
	Type     string  // photo, video
	Source   string
	Service  string
}
