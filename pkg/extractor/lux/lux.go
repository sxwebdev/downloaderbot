package lux

import (
	"context"
	"fmt"
	"strings"

	"github.com/iawia002/lux/extractors"
	"github.com/sxwebdev/downloaderbot/internal/models"

	// Import all available lux extractors to register them
	// Note: Some extractors from lux master branch are not available in v0.24.1
	_ "github.com/iawia002/lux/extractors/acfun"
	_ "github.com/iawia002/lux/extractors/bcy"
	_ "github.com/iawia002/lux/extractors/bilibili"
	_ "github.com/iawia002/lux/extractors/douyin"
	_ "github.com/iawia002/lux/extractors/douyu"
	_ "github.com/iawia002/lux/extractors/eporner"
	_ "github.com/iawia002/lux/extractors/facebook"
	_ "github.com/iawia002/lux/extractors/geekbang"
	_ "github.com/iawia002/lux/extractors/haokan"
	_ "github.com/iawia002/lux/extractors/hupu"
	_ "github.com/iawia002/lux/extractors/huya"
	_ "github.com/iawia002/lux/extractors/iqiyi"
	_ "github.com/iawia002/lux/extractors/ixigua"
	_ "github.com/iawia002/lux/extractors/kuaishou"
	_ "github.com/iawia002/lux/extractors/mgtv"
	_ "github.com/iawia002/lux/extractors/miaopai"
	_ "github.com/iawia002/lux/extractors/netease"
	_ "github.com/iawia002/lux/extractors/pinterest"
	_ "github.com/iawia002/lux/extractors/pixivision"
	_ "github.com/iawia002/lux/extractors/pornhub"
	_ "github.com/iawia002/lux/extractors/qq"
	_ "github.com/iawia002/lux/extractors/reddit"
	_ "github.com/iawia002/lux/extractors/rumble"
	_ "github.com/iawia002/lux/extractors/streamtape"
	_ "github.com/iawia002/lux/extractors/tangdou"
	_ "github.com/iawia002/lux/extractors/tiktok"
	_ "github.com/iawia002/lux/extractors/tumblr"
	_ "github.com/iawia002/lux/extractors/twitter"
	_ "github.com/iawia002/lux/extractors/udn"
	_ "github.com/iawia002/lux/extractors/vimeo"
	_ "github.com/iawia002/lux/extractors/vk"
	_ "github.com/iawia002/lux/extractors/weibo"
	_ "github.com/iawia002/lux/extractors/xiaohongshu"
	_ "github.com/iawia002/lux/extractors/ximalaya"
	_ "github.com/iawia002/lux/extractors/xinpianchang"
	_ "github.com/iawia002/lux/extractors/xvideos"
	_ "github.com/iawia002/lux/extractors/yinyuetai"
	_ "github.com/iawia002/lux/extractors/youku"
	_ "github.com/iawia002/lux/extractors/zhihu"
	_ "github.com/iawia002/lux/extractors/zingmp3"
)

// siteToSource maps lux site names to MediaSource
var siteToSource = map[string]models.MediaSource{
	"acfun":        models.MediaSourceAcfun,
	"bcy":          models.MediaSourceBcy,
	"bilibili":     models.MediaSourceBilibili,
	"douyin":       models.MediaSourceDouyin,
	"douyu":        models.MediaSourceDouyu,
	"eporner":      models.MediaSourceEporner,
	"facebook":     models.MediaSourceFacebook,
	"geekbang":     models.MediaSourceGeekbang,
	"haokan":       models.MediaSourceHaokan,
	"hupu":         models.MediaSourceHupu,
	"huya":         models.MediaSourceHuya,
	"iqiyi":        models.MediaSourceIqiyi,
	"ixigua":       models.MediaSourceXigua,
	"kuaishou":     models.MediaSourceKuaishou,
	"mgtv":         models.MediaSourceMgtv,
	"miaopai":      models.MediaSourceMiaopai,
	"netease":      models.MediaSourceNetease,
	"pinterest":    models.MediaSourcePinterest,
	"pixivision":   models.MediaSourcePixivision,
	"pornhub":      models.MediaSourcePornhub,
	"qq":           models.MediaSourceQQ,
	"reddit":       models.MediaSourceReddit,
	"rumble":       models.MediaSourceRumble,
	"streamtape":   models.MediaSourceStreamtape,
	"tangdou":      models.MediaSourceTangdou,
	"tiktok":       models.MediaSourceTikTok,
	"tumblr":       models.MediaSourceTumblr,
	"twitter":      models.MediaSourceTwitter,
	"udn":          models.MediaSourceUdn,
	"vimeo":        models.MediaSourceVimeo,
	"vk":           models.MediaSourceVK,
	"weibo":        models.MediaSourceWeibo,
	"xiaohongshu":  models.MediaSourceXiaohongshu,
	"ximalaya":     models.MediaSourceXimalaya,
	"xinpianchang": models.MediaSourceXinpianchang,
	"xvideos":      models.MediaSourceXvideos,
	"yinyuetai":    models.MediaSourceYinyuetai,
	"youku":        models.MediaSourceYouku,
	"zhihu":        models.MediaSourceZhihu,
	"zingmp3":      models.MediaSourceZingmp3,
}

// hostToSite maps host patterns to lux site names
var hostToSite = map[string]string{
	// TikTok
	"tiktok.com":        "tiktok",
	"vm.tiktok.com":     "tiktok",
	"vt.tiktok.com":     "tiktok",
	// Twitter/X
	"twitter.com":       "twitter",
	"x.com":             "twitter",
	"mobile.twitter.com": "twitter",
	// Facebook
	"facebook.com":      "facebook",
	"fb.watch":          "facebook",
	"fb.com":            "facebook",
	// Vimeo
	"vimeo.com":         "vimeo",
	"player.vimeo.com":  "vimeo",
	// Reddit
	"reddit.com":        "reddit",
	"old.reddit.com":    "reddit",
	"redd.it":           "reddit",
	// Pinterest
	"pinterest.com":     "pinterest",
	"pin.it":            "pinterest",
	// Tumblr
	"tumblr.com":        "tumblr",
	// Bilibili
	"bilibili.com":      "bilibili",
	"b23.tv":            "bilibili",
	// Douyin
	"douyin.com":        "douyin",
	"iesdouyin.com":     "douyin",
	// Weibo
	"weibo.com":         "weibo",
	"weibo.cn":          "weibo",
	// Xiaohongshu
	"xiaohongshu.com":   "xiaohongshu",
	"xhslink.com":       "xiaohongshu",
	// VK
	"vk.com":            "vk",
	"vkvideo.ru":        "vk",
	// Rumble
	"rumble.com":        "rumble",
	// iQIYI
	"iqiyi.com":         "iqiyi",
	// Youku
	"youku.com":         "youku",
	// Kuaishou
	"kuaishou.com":      "kuaishou",
	// Xigua
	"ixigua.com":        "ixigua",
	// Douyu
	"douyu.com":         "douyu",
	// Huya
	"huya.com":          "huya",
	// MGTV
	"mgtv.com":          "mgtv",
	// QQ
	"qq.com":            "qq",
	"v.qq.com":          "qq",
	// Netease
	"163.com":           "netease",
	"music.163.com":     "netease",
	// Zhihu
	"zhihu.com":         "zhihu",
	// Acfun
	"acfun.cn":          "acfun",
	// BCY
	"bcy.net":           "bcy",
	// Eporner
	"eporner.com":       "eporner",
	// Geekbang
	"geekbang.org":      "geekbang",
	// Haokan
	"haokan.baidu.com":  "haokan",
	// Hupu
	"hupu.com":          "hupu",
	// Miaopai
	"miaopai.com":       "miaopai",
	// Pixivision
	"pixivision.net":    "pixivision",
	// Pornhub
	"pornhub.com":       "pornhub",
	// Streamtape
	"streamtape.com":    "streamtape",
	// Tangdou
	"tangdou.com":       "tangdou",
	// UDN
	"udn.com":           "udn",
	"video.udn.com":     "udn",
	// Ximalaya
	"ximalaya.com":      "ximalaya",
	// Xinpianchang
	"xinpianchang.com":  "xinpianchang",
	// Xvideos
	"xvideos.com":       "xvideos",
	// Yinyuetai
	"yinyuetai.com":     "yinyuetai",
	// Zingmp3
	"zingmp3.vn":        "zingmp3",
}

// Extractor implements the extractor.Extractor interface using lux library
type Extractor struct {
	name  string
	hosts []string
}

// NewExtractor creates a new lux-based extractor for a specific site
func NewExtractor(siteName string, hosts []string) *Extractor {
	return &Extractor{
		name:  siteName,
		hosts: hosts,
	}
}

// Name returns the extractor name
func (e *Extractor) Name() string {
	return e.name
}

// Hosts returns the supported hosts
func (e *Extractor) Hosts() []string {
	return e.hosts
}

// Extract extracts media from the URL using lux
func (e *Extractor) Extract(ctx context.Context, url string) (*models.Media, error) {
	opts := extractors.Options{}

	dataList, err := extractors.Extract(url, opts)
	if err != nil {
		return nil, fmt.Errorf("lux extraction failed: %w", err)
	}

	if len(dataList) == 0 {
		return nil, fmt.Errorf("no data extracted from URL")
	}

	// Use the first data item
	data := dataList[0]
	if data.Err != nil {
		return nil, fmt.Errorf("extraction error: %w", data.Err)
	}

	// Convert lux data to our Media model
	media := convertLuxDataToMedia(data, url)

	// Set the source based on the extractor name
	if source, ok := siteToSource[e.name]; ok {
		media.Source = source
	} else {
		media.Source = models.MediaSource(e.name)
	}

	return media, nil
}

// convertLuxDataToMedia converts lux Data to our Media model
func convertLuxDataToMedia(data *extractors.Data, requestURL string) *models.Media {
	media := &models.Media{
		RequestUrl: requestURL,
		Title:      data.Title,
		Items:      make([]*models.MediaItem, 0),
	}

	// Iterate over streams to get quality options
	for streamID, stream := range data.Streams {
		if stream == nil || len(stream.Parts) == 0 {
			continue
		}

		// Determine media type based on lux data type
		mediaType := models.MediaTypeVideo
		switch data.Type {
		case extractors.DataTypeAudio:
			mediaType = models.MediaTypeAudio
		case extractors.DataTypeImage:
			mediaType = models.MediaTypePhoto
		}

		// For each part in the stream, create a media item
		for partIdx, part := range stream.Parts {
			if part == nil || part.URL == "" {
				continue
			}

			quality := stream.Quality
			if quality == "" {
				quality = streamID
			}
			if len(stream.Parts) > 1 {
				quality = fmt.Sprintf("%s-part%d", quality, partIdx+1)
			}

			item := &models.MediaItem{
				Id:            fmt.Sprintf("%s-%d", streamID, partIdx),
				Type:          mediaType,
				Url:           part.URL,
				Quality:       quality,
				ContentLength: part.Size,
				MimeType:      getMimeType(part.Ext, mediaType),
			}

			media.Items = append(media.Items, item)
		}
	}

	// If no items from streams, try to get any available data
	if len(media.Items) == 0 && data.URL != "" {
		media.Items = append(media.Items, &models.MediaItem{
			Type: models.MediaTypeVideo,
			Url:  data.URL,
		})
	}

	return media
}

// getMimeType returns MIME type based on extension and media type
func getMimeType(ext string, mediaType models.MediaType) string {
	ext = strings.TrimPrefix(ext, ".")
	switch mediaType {
	case models.MediaTypeAudio:
		switch ext {
		case "mp3":
			return "audio/mpeg"
		case "m4a", "aac":
			return "audio/mp4"
		case "ogg":
			return "audio/ogg"
		case "flac":
			return "audio/flac"
		case "wav":
			return "audio/wav"
		default:
			return "audio/mpeg"
		}
	case models.MediaTypePhoto:
		switch ext {
		case "jpg", "jpeg":
			return "image/jpeg"
		case "png":
			return "image/png"
		case "gif":
			return "image/gif"
		case "webp":
			return "image/webp"
		default:
			return "image/jpeg"
		}
	default:
		switch ext {
		case "mp4":
			return "video/mp4"
		case "webm":
			return "video/webm"
		case "mkv":
			return "video/x-matroska"
		case "flv":
			return "video/x-flv"
		case "m4v":
			return "video/mp4"
		default:
			return "video/mp4"
		}
	}
}

// GetAllExtractors returns all available lux-based extractors
func GetAllExtractors() []*Extractor {
	// Group hosts by site
	siteHosts := make(map[string][]string)
	for host, site := range hostToSite {
		siteHosts[site] = append(siteHosts[site], host)
	}

	extractors := make([]*Extractor, 0, len(siteHosts))
	for site, hosts := range siteHosts {
		extractors = append(extractors, NewExtractor(site, hosts))
	}

	return extractors
}

// GetExtractorBySite returns an extractor for a specific site
func GetExtractorBySite(siteName string) *Extractor {
	var hosts []string
	for host, site := range hostToSite {
		if site == siteName {
			hosts = append(hosts, host)
		}
	}
	if len(hosts) == 0 {
		return nil
	}
	return NewExtractor(siteName, hosts)
}
