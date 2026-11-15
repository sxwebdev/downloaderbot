package lux

import (
	"context"
	"fmt"
	"strings"

	"github.com/iawia002/lux/extractors"
	"github.com/iawia002/lux/extractors/acfun"
	"github.com/iawia002/lux/extractors/bcy"
	"github.com/iawia002/lux/extractors/bilibili"
	"github.com/iawia002/lux/extractors/douyin"
	"github.com/iawia002/lux/extractors/douyu"
	"github.com/iawia002/lux/extractors/eporner"
	"github.com/iawia002/lux/extractors/facebook"
	"github.com/iawia002/lux/extractors/geekbang"
	"github.com/iawia002/lux/extractors/haokan"
	"github.com/iawia002/lux/extractors/hupu"
	"github.com/iawia002/lux/extractors/huya"
	"github.com/iawia002/lux/extractors/iqiyi"
	"github.com/iawia002/lux/extractors/ixigua"
	"github.com/iawia002/lux/extractors/kuaishou"
	"github.com/iawia002/lux/extractors/mgtv"
	"github.com/iawia002/lux/extractors/miaopai"
	"github.com/iawia002/lux/extractors/netease"
	"github.com/iawia002/lux/extractors/pinterest"
	"github.com/iawia002/lux/extractors/pixivision"
	"github.com/iawia002/lux/extractors/pornhub"
	"github.com/iawia002/lux/extractors/qq"
	"github.com/iawia002/lux/extractors/reddit"
	"github.com/iawia002/lux/extractors/rumble"
	"github.com/iawia002/lux/extractors/streamtape"
	"github.com/iawia002/lux/extractors/tangdou"
	"github.com/iawia002/lux/extractors/tiktok"
	"github.com/iawia002/lux/extractors/tumblr"
	"github.com/iawia002/lux/extractors/twitter"
	"github.com/iawia002/lux/extractors/udn"
	"github.com/iawia002/lux/extractors/vimeo"
	"github.com/iawia002/lux/extractors/vk"
	"github.com/iawia002/lux/extractors/weibo"
	"github.com/iawia002/lux/extractors/xiaohongshu"
	"github.com/iawia002/lux/extractors/ximalaya"
	"github.com/iawia002/lux/extractors/xinpianchang"
	"github.com/iawia002/lux/extractors/xvideos"
	"github.com/iawia002/lux/extractors/yinyuetai"
	"github.com/iawia002/lux/extractors/youku"
	"github.com/iawia002/lux/extractors/zhihu"
	"github.com/iawia002/lux/extractors/zingmp3"
	"github.com/sxwebdev/downloaderbot/internal/models"
)

// siteExtractors builds the lux extractor for each site name. Extract calls it
// directly instead of going through extractors.Extract: lux picks its extractor
// from the URL host's second-level label, so short-link hosts our registry maps
// to a site — pin.it, x.com, fb.watch, redd.it, vkvideo.ru — resolve to keys
// lux never registered ("pin", "x", ...). lux then falls back to its universal
// extractor, which is not linked in, and calls a method on a nil interface.
var siteExtractors = map[string]func() extractors.Extractor{
	"acfun":        acfun.New,
	"bcy":          bcy.New,
	"bilibili":     bilibili.New,
	"douyin":       douyin.New,
	"douyu":        douyu.New,
	"eporner":      eporner.New,
	"facebook":     facebook.New,
	"geekbang":     geekbang.New,
	"haokan":       haokan.New,
	"hupu":         hupu.New,
	"huya":         huya.New,
	"iqiyi":        func() extractors.Extractor { return iqiyi.New(iqiyi.SiteTypeIqiyi) },
	"ixigua":       ixigua.New,
	"kuaishou":     kuaishou.New,
	"mgtv":         mgtv.New,
	"miaopai":      miaopai.New,
	"netease":      netease.New,
	"pinterest":    pinterest.New,
	"pixivision":   pixivision.New,
	"pornhub":      pornhub.New,
	"qq":           qq.New,
	"reddit":       reddit.New,
	"rumble":       rumble.New,
	"streamtape":   streamtape.New,
	"tangdou":      tangdou.New,
	"tiktok":       tiktok.New,
	"tumblr":       tumblr.New,
	"twitter":      twitter.New,
	"udn":          udn.New,
	"vimeo":        vimeo.New,
	"vk":           vk.New,
	"weibo":        weibo.New,
	"xiaohongshu":  xiaohongshu.New,
	"ximalaya":     ximalaya.New,
	"xinpianchang": xinpianchang.New,
	"xvideos":      xvideos.New,
	"yinyuetai":    yinyuetai.New,
	"youku":        youku.New,
	"zhihu":        zhihu.New,
	"zingmp3":      zingmp3.New,
}

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
	// TikTok is handled by the custom rod-based extractor (pkg/extractor/tiktok),
	// because lux's TikTok extractor is blocked by anti-bot. Do not map it here.
	// Twitter/X (note: `mobile.` and `www.` are normalized away by the registry)
	"twitter.com": "twitter",
	"x.com":       "twitter",
	// Facebook
	"facebook.com": "facebook",
	"fb.watch":     "facebook",
	"fb.com":       "facebook",
	// Vimeo
	"vimeo.com":        "vimeo",
	"player.vimeo.com": "vimeo",
	// Reddit
	"reddit.com":     "reddit",
	"old.reddit.com": "reddit",
	"redd.it":        "reddit",
	// Pinterest
	"pinterest.com": "pinterest",
	"pin.it":        "pinterest",
	// Tumblr
	"tumblr.com": "tumblr",
	// Bilibili
	"bilibili.com": "bilibili",
	"b23.tv":       "bilibili",
	// Douyin
	"douyin.com":    "douyin",
	"iesdouyin.com": "douyin",
	// Weibo
	"weibo.com": "weibo",
	"weibo.cn":  "weibo",
	// Xiaohongshu
	"xiaohongshu.com": "xiaohongshu",
	"xhslink.com":     "xiaohongshu",
	// VK
	"vk.com":     "vk",
	"vkvideo.ru": "vk",
	// Rumble
	"rumble.com": "rumble",
	// iQIYI
	"iqiyi.com": "iqiyi",
	// Youku
	"youku.com": "youku",
	// Kuaishou
	"kuaishou.com": "kuaishou",
	// Xigua
	"ixigua.com": "ixigua",
	// Douyu
	"douyu.com": "douyu",
	// Huya
	"huya.com": "huya",
	// MGTV
	"mgtv.com": "mgtv",
	// QQ
	"qq.com":   "qq",
	"v.qq.com": "qq",
	// Netease
	"163.com":       "netease",
	"music.163.com": "netease",
	// Zhihu
	"zhihu.com": "zhihu",
	// Acfun
	"acfun.cn": "acfun",
	// BCY
	"bcy.net": "bcy",
	// Eporner
	"eporner.com": "eporner",
	// Geekbang
	"geekbang.org": "geekbang",
	// Haokan
	"haokan.baidu.com": "haokan",
	// Hupu
	"hupu.com": "hupu",
	// Miaopai
	"miaopai.com": "miaopai",
	// Pixivision
	"pixivision.net": "pixivision",
	// Pornhub
	"pornhub.com": "pornhub",
	// Streamtape
	"streamtape.com": "streamtape",
	// Tangdou
	"tangdou.com": "tangdou",
	// UDN
	"udn.com":       "udn",
	"video.udn.com": "udn",
	// Ximalaya
	"ximalaya.com": "ximalaya",
	// Xinpianchang
	"xinpianchang.com": "xinpianchang",
	// Xvideos
	"xvideos.com": "xvideos",
	// Yinyuetai
	"yinyuetai.com": "yinyuetai",
	// Zingmp3
	"zingmp3.vn": "zingmp3",
}

// Extractor implements the extractor.Extractor interface using lux library
type Extractor struct {
	name  string
	hosts []string
	lux   extractors.Extractor // nil when lux has no extractor for the site
}

// NewExtractor creates a new lux-based extractor for a specific site
func NewExtractor(siteName string, hosts []string) *Extractor {
	e := &Extractor{
		name:  siteName,
		hosts: hosts,
	}
	if newLux, ok := siteExtractors[siteName]; ok {
		e.lux = newLux()
	}
	return e
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
	dataList, err := e.extractLux(strings.TrimSpace(url))
	if err != nil {
		return nil, fmt.Errorf("lux extraction failed: %w", err)
	}

	if len(dataList) == 0 || dataList[0] == nil {
		return nil, fmt.Errorf("no data extracted from URL")
	}

	// Use the first data item
	data := dataList[0]
	if data.Err != nil {
		return nil, fmt.Errorf("extraction error: %w", data.Err)
	}

	// Convert lux data to our Media model
	media := convertLuxDataToMedia(data, url)
	if len(media.Items) == 0 {
		return nil, fmt.Errorf("no downloadable media: %s only offers segmented streams that need merging", e.name)
	}

	// Set the source based on the extractor name
	if source, ok := siteToSource[e.name]; ok {
		media.Source = source
	} else {
		media.Source = models.MediaSource(e.name)
	}

	return media, nil
}

// extractLux runs the site's lux extractor. lux scrapes arbitrary third-party
// pages and indexes into what it finds, so a panic in it is turned into an
// error for this one link rather than left to the Telegram handler.
func (e *Extractor) extractLux(url string) (data []*extractors.Data, err error) {
	if e.lux == nil {
		return nil, fmt.Errorf("no lux extractor for site %q", e.name)
	}
	defer func() {
		if r := recover(); r != nil {
			data, err = nil, fmt.Errorf("lux %s extractor panicked: %v", e.name, r)
		}
	}()
	return e.lux.Extract(url, extractors.Options{})
}

// convertLuxDataToMedia converts lux Data to our Media model
func convertLuxDataToMedia(data *extractors.Data, requestURL string) *models.Media {
	media := &models.Media{
		RequestUrl: requestURL,
		Title:      data.Title,
		Items:      make([]*models.MediaItem, 0),
	}

	// Determine media type based on lux data type
	mediaType := models.MediaTypeVideo
	switch data.Type {
	case extractors.DataTypeAudio:
		mediaType = models.MediaTypeAudio
	case extractors.DataTypeImage:
		mediaType = models.MediaTypePhoto
	}

	// One item per quality. Only single-part streams are usable: lux splits
	// HLS playlists (Twitter/X) into their segments and DASH (Bilibili) into
	// separate video and audio tracks, which lux itself merges with ffmpeg.
	// Sent as-is they would reach the user as an album of 2-second fragments.
	for streamID, stream := range data.Streams {
		if stream == nil || len(stream.Parts) != 1 {
			continue
		}
		part := stream.Parts[0]
		if part == nil || part.URL == "" {
			continue
		}

		quality := stream.Quality
		if quality == "" {
			quality = streamID
		}

		media.Items = append(media.Items, &models.MediaItem{
			Id:            streamID + "-0",
			Type:          mediaType,
			Url:           part.URL,
			Quality:       quality,
			ContentLength: part.Size,
			MimeType:      getMimeType(part.Ext, mediaType),
		})
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
