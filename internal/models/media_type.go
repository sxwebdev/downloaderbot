package models

const (
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
	MediaTypePhoto MediaType = "photo"
)

type MediaType string

func (s MediaType) Valid() bool {
	switch s {
	case MediaTypeAudio,
		MediaTypeVideo,
		MediaTypePhoto:
		return true
	default:
		return false
	}
}

func (s MediaType) IsAudio() bool {
	return s == MediaTypeAudio
}

func (s MediaType) IsVideo() bool {
	return s == MediaTypeVideo
}

func (s MediaType) IsPhone() bool {
	return s == MediaTypePhoto
}
