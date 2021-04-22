package channels

type InfoChannelsUS915 struct {
	ListChanLastPass [8]int `json:"-"`
	Pass     int    `json:"-"`
}
