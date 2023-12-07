package models

type MediaSource string

func (s MediaSource) Valid() bool {
	switch s {
	case MediaSourceInstagram, MediaSourceYoutube:
		return true
	default:
		return false
	}
}

const (
	MediaSourceInstagram MediaSource = "instagram"
	MediaSourceYoutube   MediaSource = "youtube"
)
