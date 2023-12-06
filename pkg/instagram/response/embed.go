package response

// Dimensions of the media
type Dimensions struct {
	Height int `json:"height"` // Height of the media in pixels
	Width  int `json:"width"`  // Width of the media in pixels
}

// Resource for an specific Media
type Resource struct {
	Width  int    `json:"config_width"`  // Height of the resource media in pixels
	Height int    `json:"config_height"` // Height of the resource media in pixels
	Src    string `json:"src"`           // Direct URL to the Resource
}

// Caption contains the raw caption of the Instagram post
type Caption struct {
	// List of Edge which contains multiple nodes
	Edges []struct {
		// A single node which contains Caption text
		Node struct {
			Text string `json:"text"` // The raw caption text
		} `json:"node"`
	} `json:"edges"`
}

// WithCount represents a single numeric Count field
type WithCount struct {
	Count uint64 `json:"count"` // The number of count
}

// OwnerTimeline contains the latest timeline feed of the Instagram user
type OwnerTimeline struct {
	Count uint64 `json:"count"` // Count of public posts
	// List of Edge which contains multiple nodes
	Edges []struct {
		// A single node which contains Thumbnail
		Node struct {
			Id                 string     `json:"string"`
			ThumbnailSrc       string     `json:"thumbnail_src"`
			ThumbnailResources []Resource `json:"thumbnail_resources"`
		} `json:"node"`
	} `json:"edges"`
}

// Owner is a single Instagram user who owns the Media
type Owner struct {
	Id                string        `json:"id"`                           // Unique ID of the User
	ProfilePictureURL string        `json:"profile_pic_url"`              // URL of user's profile picture
	Username          string        `json:"username"`                     // Username of the User
	HasPublicStory    bool          `json:"has_public_story"`             // Is the User stories publically available
	IsPrivate         bool          `json:"is_private"`                   // Is the User account private
	IsVerified        bool          `json:"is_verified"`                  // Is the User account verified
	Followers         WithCount     `json:"edge_followed_by"`             // Followers count
	Timeline          OwnerTimeline `json:"edge_owner_to_timeline_media"` // Timeline feeds of the User
}

// SliderItemNode contains information about the Instagram post
type SliderItemNode struct {
	Id               string     `json:"id"`                // Unique ID of the Media
	Shortcode        string     `json:"shortcode"`         // Unique shortcode of the Media
	Type             string     `json:"__typename"`        // Type of the Media
	ProductType      string     `json:"product_type"`      // Product type of the Media
	Dimensions       Dimensions `json:"dimensions"`        // Dimension of the Media
	DisplayURL       string     `json:"display_url"`       // URL of the Media (resolution is dynamic)
	DisplayResources []Resource `json:"display_resources"` // Resource of the Media

	IsVideo        bool   `json:"is_video"`         // Is type of the Media equals to video
	Title          string `json:"title"`            // The video title
	VideoURL       string `json:"video_url"`        // Direct URL to the Video
	VideoViewCount uint64 `json:"video_view_count"` // The number of times Video has been viewed

	// clips_music_attribution_info
	// media_overlay_info
	// sharing_friction_info
}

// ExtractMediaURL will extract the Media URL automatically based on Media type (video or image)
func (s SliderItemNode) ExtractMediaURL() string {
	if s.IsVideo {
		return s.VideoURL
	}
	return s.DisplayURL
}

// SliderItems contains list of the Instagram posts
type SliderItems struct {
	// List of Edge which contains multiple nodes
	Edges []struct {
		// A single node which contains Media item
		Node SliderItemNode `json:"node"`
	} `json:"edges"`
}

// Media which contains a single Instagram post
type Media struct {
	Id               string     `json:"id"`                    // Unique ID of the Media
	Shortcode        string     `json:"shortcode"`             // Unique shortcode of the Media
	Type             string     `json:"__typename"`            // Type of the Media
	ProductType      string     `json:"product_type"`          // Product type of the Media
	TakenAt          Time       `json:"taken_at_timestamp"`    // The time this media was taken/published
	CommenterCount   uint64     `json:"commenter_count"`       // Count of Users who have commented
	Comments         WithCount  `json:"edge_media_to_comment"` // Comments count
	Likes            WithCount  `json:"edge_liked_by"`         // Likes count
	Dimensions       Dimensions `json:"dimensions"`            // Dimensions of the Media
	DisplayURL       string     `json:"display_url"`           // URL of the Media (resolution is dynamic)
	DisplayResources []Resource `json:"display_resources"`     // Resource of the Media

	Caption Caption `json:"edge_media_to_caption"` // Media caption
	Owner   Owner   `json:"owner"`                 // User who has posted this Media

	IsVideo        bool   `json:"is_video"`         // Is type of the Media equals to video
	Title          string `json:"title"`            // The video title
	VideoURL       string `json:"video_url"`        // Direct URL to the Video
	VideoViewCount uint64 `json:"video_view_count"` // The number of times Video has been viewed

	SliderItems SliderItems `json:"edge_sidecar_to_children"` // Children of the Media

	// clips_music_attribution_info
	// media_overlay_info
	// sharing_friction_info

	// edge_media_to_sponsor_user
	// is_affiliate
	// is_paid_partnership
	// location
	// coauthor_producers
}

// EmbedResponse base
type EmbedResponse struct {
	Media Media `json:"shortcode_media"` // Media
}

// IsEmpty will return true if the Media object is empty
func (s EmbedResponse) IsEmpty() bool {
	return s.Media.Id == ""
}

// IsVideo will return true if the Media type is equals to video
func (s EmbedResponse) IsVideo() bool {
	return s.Media.IsVideo
}

// GetCaption of the Media
func (s EmbedResponse) GetCaption() string {
	if len(s.Media.Caption.Edges) > 0 {
		return s.Media.Caption.Edges[0].Node.Text
	}

	return s.Media.Title
}

// ExtractMediaURL will extract the Media URL automatically based on Media type (video or image)
func (s EmbedResponse) ExtractMediaURL() string {
	if s.Media.IsVideo {
		return s.Media.VideoURL
	}
	return s.Media.DisplayURL
}
