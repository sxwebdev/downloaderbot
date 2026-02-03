package models

type MediaSource string

func (s MediaSource) String() string {
	return string(s)
}

// Common media sources
const (
	MediaSourceInstagram    MediaSource = "instagram"
	MediaSourceYoutube      MediaSource = "youtube"
	MediaSourceTikTok       MediaSource = "tiktok"
	MediaSourceTwitter      MediaSource = "twitter"
	MediaSourceFacebook     MediaSource = "facebook"
	MediaSourceVimeo        MediaSource = "vimeo"
	MediaSourceReddit       MediaSource = "reddit"
	MediaSourcePinterest    MediaSource = "pinterest"
	MediaSourceTumblr       MediaSource = "tumblr"
	MediaSourceBilibili     MediaSource = "bilibili"
	MediaSourceDouyin       MediaSource = "douyin"
	MediaSourceWeibo        MediaSource = "weibo"
	MediaSourceXiaohongshu  MediaSource = "xiaohongshu"
	MediaSourceVK           MediaSource = "vk"
	MediaSourceRumble       MediaSource = "rumble"
	MediaSourceIqiyi        MediaSource = "iqiyi"
	MediaSourceYouku        MediaSource = "youku"
	MediaSourceKuaishou     MediaSource = "kuaishou"
	MediaSourceXigua        MediaSource = "ixigua"
	MediaSourceDouyu        MediaSource = "douyu"
	MediaSourceHuya         MediaSource = "huya"
	MediaSourceMgtv         MediaSource = "mgtv"
	MediaSourceQQ           MediaSource = "qq"
	MediaSourceNetease      MediaSource = "netease"
	MediaSourceZhihu        MediaSource = "zhihu"
	MediaSourceAcfun        MediaSource = "acfun"
	MediaSourceBcy          MediaSource = "bcy"
	MediaSourceEporner      MediaSource = "eporner"
	MediaSourceGeekbang     MediaSource = "geekbang"
	MediaSourceHaokan       MediaSource = "haokan"
	MediaSourceHupu         MediaSource = "hupu"
	MediaSourceMiaopai      MediaSource = "miaopai"
	MediaSourcePixivision   MediaSource = "pixivision"
	MediaSourcePornhub      MediaSource = "pornhub"
	MediaSourceStreamtape   MediaSource = "streamtape"
	MediaSourceTangdou      MediaSource = "tangdou"
	MediaSourceUdn          MediaSource = "udn"
	MediaSourceXimalaya     MediaSource = "ximalaya"
	MediaSourceXinpianchang MediaSource = "xinpianchang"
	MediaSourceXvideos      MediaSource = "xvideos"
	MediaSourceYinyuetai    MediaSource = "yinyuetai"
	MediaSourceZingmp3      MediaSource = "zingmp3"
)
