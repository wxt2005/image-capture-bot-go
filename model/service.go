package model

type ImageService interface {
	ExtractMedias(urls []string) ([]*Media, []string, error)
}

type ConsumerService interface {
	ConsumeMedias(medias []*Media)
}
