package response

type GrpahSQLResponse struct {
	Data struct {
		XdtShortcodeMedia struct {
			Typename                 string `json:"__typename"`
			IsXDTGraphMediaInterface string `json:"__isXDTGraphMediaInterface"`
			ID                       string `json:"id"`
			Shortcode                string `json:"shortcode"`
			ThumbnailSrc             string `json:"thumbnail_src"`
			Dimensions               struct {
				Height int `json:"height"`
				Width  int `json:"width"`
			} `json:"dimensions"`
			GatingInfo              interface{} `json:"gating_info"`
			FactCheckOverallRating  interface{} `json:"fact_check_overall_rating"`
			FactCheckInformation    interface{} `json:"fact_check_information"`
			SensitivityFrictionInfo interface{} `json:"sensitivity_friction_info"`
			SharingFrictionInfo     struct {
				ShouldHaveSharingFriction bool        `json:"should_have_sharing_friction"`
				BloksAppURL               interface{} `json:"bloks_app_url"`
			} `json:"sharing_friction_info"`
			MediaOverlayInfo interface{} `json:"media_overlay_info"`
			MediaPreview     string      `json:"media_preview"`
			DisplayURL       string      `json:"display_url"`
			DisplayResources []struct {
				Src          string `json:"src"`
				ConfigWidth  int    `json:"config_width"`
				ConfigHeight int    `json:"config_height"`
			} `json:"display_resources"`
			AccessibilityCaption interface{} `json:"accessibility_caption"`
			DashInfo             struct {
				IsDashEligible    bool        `json:"is_dash_eligible"`
				VideoDashManifest interface{} `json:"video_dash_manifest"`
				NumberOfQualities int         `json:"number_of_qualities"`
			} `json:"dash_info"`
			HasAudio                  bool        `json:"has_audio"`
			VideoURL                  string      `json:"video_url"`
			VideoViewCount            int         `json:"video_view_count"`
			VideoPlayCount            int         `json:"video_play_count"`
			EncodingStatus            interface{} `json:"encoding_status"`
			IsPublished               bool        `json:"is_published"`
			ProductType               string      `json:"product_type"`
			Title                     string      `json:"title"`
			VideoDuration             float64     `json:"video_duration"`
			ClipsMusicAttributionInfo struct {
				ArtistName            string `json:"artist_name"`
				SongName              string `json:"song_name"`
				UsesOriginalAudio     bool   `json:"uses_original_audio"`
				ShouldMuteAudio       bool   `json:"should_mute_audio"`
				ShouldMuteAudioReason string `json:"should_mute_audio_reason"`
				AudioID               string `json:"audio_id"`
			} `json:"clips_music_attribution_info"`
			IsVideo               bool        `json:"is_video"`
			TrackingToken         string      `json:"tracking_token"`
			UpcomingEvent         interface{} `json:"upcoming_event"`
			EdgeMediaToTaggedUser struct {
				Edges []struct {
					Node struct {
						User struct {
							FullName         string `json:"full_name"`
							FollowedByViewer bool   `json:"followed_by_viewer"`
							ID               string `json:"id"`
							IsVerified       bool   `json:"is_verified"`
							ProfilePicURL    string `json:"profile_pic_url"`
							Username         string `json:"username"`
						} `json:"user"`
						X  int    `json:"x"`
						Y  int    `json:"y"`
						ID string `json:"id"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_media_to_tagged_user"`
			Owner struct {
				ID                        string      `json:"id"`
				Username                  string      `json:"username"`
				IsVerified                bool        `json:"is_verified"`
				ProfilePicURL             string      `json:"profile_pic_url"`
				BlockedByViewer           bool        `json:"blocked_by_viewer"`
				RestrictedByViewer        interface{} `json:"restricted_by_viewer"`
				FollowedByViewer          bool        `json:"followed_by_viewer"`
				FullName                  string      `json:"full_name"`
				HasBlockedViewer          bool        `json:"has_blocked_viewer"`
				IsEmbedsDisabled          bool        `json:"is_embeds_disabled"`
				IsPrivate                 bool        `json:"is_private"`
				IsUnpublished             bool        `json:"is_unpublished"`
				RequestedByViewer         bool        `json:"requested_by_viewer"`
				PassTieringRecommendation bool        `json:"pass_tiering_recommendation"`
				EdgeOwnerToTimelineMedia  struct {
					Count int `json:"count"`
				} `json:"edge_owner_to_timeline_media"`
				EdgeFollowedBy struct {
					Count int `json:"count"`
				} `json:"edge_followed_by"`
			} `json:"owner"`
			EdgeMediaToCaption struct {
				Edges []struct {
					Node struct {
						CreatedAt string `json:"created_at"`
						Text      string `json:"text"`
						ID        string `json:"id"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_media_to_caption"`
			CanSeeInsightsAsBrand     bool `json:"can_see_insights_as_brand"`
			CaptionIsEdited           bool `json:"caption_is_edited"`
			HasRankedComments         bool `json:"has_ranked_comments"`
			LikeAndViewCountsDisabled bool `json:"like_and_view_counts_disabled"`
			EdgeMediaToParentComment  struct {
				Count    int `json:"count"`
				PageInfo struct {
					HasNextPage bool        `json:"has_next_page"`
					EndCursor   interface{} `json:"end_cursor"`
				} `json:"page_info"`
				Edges []struct {
					Node struct {
						ID              string `json:"id"`
						Text            string `json:"text"`
						CreatedAt       int    `json:"created_at"`
						DidReportAsSpam bool   `json:"did_report_as_spam"`
						Owner           struct {
							ID            string `json:"id"`
							IsVerified    bool   `json:"is_verified"`
							ProfilePicURL string `json:"profile_pic_url"`
							Username      string `json:"username"`
						} `json:"owner"`
						ViewerHasLiked bool `json:"viewer_has_liked"`
						EdgeLikedBy    struct {
							Count int `json:"count"`
						} `json:"edge_liked_by"`
						IsRestrictedPending  bool `json:"is_restricted_pending"`
						EdgeThreadedComments struct {
							Count    int `json:"count"`
							PageInfo struct {
								HasNextPage bool        `json:"has_next_page"`
								EndCursor   interface{} `json:"end_cursor"`
							} `json:"page_info"`
							Edges []interface{} `json:"edges"`
						} `json:"edge_threaded_comments"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_media_to_parent_comment"`
			EdgeMediaToHoistedComment struct {
				Edges []interface{} `json:"edges"`
			} `json:"edge_media_to_hoisted_comment"`
			EdgeMediaPreviewComment struct {
				Count int `json:"count"`
				Edges []struct {
					Node struct {
						ID              string `json:"id"`
						Text            string `json:"text"`
						CreatedAt       int    `json:"created_at"`
						DidReportAsSpam bool   `json:"did_report_as_spam"`
						Owner           struct {
							ID            string `json:"id"`
							IsVerified    bool   `json:"is_verified"`
							ProfilePicURL string `json:"profile_pic_url"`
							Username      string `json:"username"`
						} `json:"owner"`
						ViewerHasLiked bool `json:"viewer_has_liked"`
						EdgeLikedBy    struct {
							Count int `json:"count"`
						} `json:"edge_liked_by"`
						IsRestrictedPending bool `json:"is_restricted_pending"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_media_preview_comment"`
			CommentsDisabled            bool `json:"comments_disabled"`
			CommentingDisabledForViewer bool `json:"commenting_disabled_for_viewer"`
			TakenAtTimestamp            int  `json:"taken_at_timestamp"`
			EdgeMediaPreviewLike        struct {
				Count int           `json:"count"`
				Edges []interface{} `json:"edges"`
			} `json:"edge_media_preview_like"`
			EdgeMediaToSponsorUser struct {
				Edges []interface{} `json:"edges"`
			} `json:"edge_media_to_sponsor_user"`
			IsAffiliate                bool        `json:"is_affiliate"`
			IsPaidPartnership          bool        `json:"is_paid_partnership"`
			Location                   interface{} `json:"location"`
			NftAssetInfo               interface{} `json:"nft_asset_info"`
			ViewerHasLiked             bool        `json:"viewer_has_liked"`
			ViewerHasSaved             bool        `json:"viewer_has_saved"`
			ViewerHasSavedToCollection bool        `json:"viewer_has_saved_to_collection"`
			ViewerInPhotoOfYou         bool        `json:"viewer_in_photo_of_you"`
			ViewerCanReshare           bool        `json:"viewer_can_reshare"`
			IsAd                       bool        `json:"is_ad"`
			EdgeWebMediaToRelatedMedia struct {
				Edges []interface{} `json:"edges"`
			} `json:"edge_web_media_to_related_media"`
			CoauthorProducers []struct {
				ID            string `json:"id"`
				IsVerified    bool   `json:"is_verified"`
				ProfilePicURL string `json:"profile_pic_url"`
				Username      string `json:"username"`
			} `json:"coauthor_producers"`
			PinnedForUsers      []interface{} `json:"pinned_for_users"`
			EdgeRelatedProfiles struct {
				Edges []interface{} `json:"edges"`
			} `json:"edge_related_profiles"`
		} `json:"xdt_shortcode_media"`
	} `json:"data"`
	Extensions struct {
		IsFinal bool `json:"is_final"`
	} `json:"extensions"`
}
