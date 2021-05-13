package channels

type InfoChannelsUS915 struct {
	ListChannelsLastPass [8]int `json:"-"`
	FirstPass            bool   `json:"-"`
}
