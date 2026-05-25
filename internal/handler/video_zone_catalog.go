package handler

// videoZoneCatalog mirrors cakecake-vue/src/constants/videoZones.js (menuLeft 分区).
var videoZoneCatalog = []struct {
	parent string
	subs   []string
}{
	{"动画", []string{"MAD·AMV", "MMD·3D", "短片·手书·配音", "综合"}},
	{"番剧", []string{"连载动画", "完结动画", "资讯", "官方延伸", "新番时间表", "番剧索引"}},
	{"国创", []string{"国产动画", "国产原创相关", "布袋戏", "资讯", "新番时间表", "国产动画索引"}},
	{"音乐", []string{"原创音乐", "翻唱", "VOCALOID·UTAU", "演奏", "三次元音乐", "OP/ED/OST", "音乐选集"}},
	{"舞蹈", []string{"宅舞", "三次元舞蹈", "舞蹈教程"}},
	{"游戏", []string{"单机游戏", "电子竞技", "手机游戏", "网络游戏", "桌游棋牌", "GMV", "音游", "Mugen"}},
	{"科技", []string{"趣味科普人文", "野生技术协会", "演讲·公开课", "星海", "数码", "机械", "汽车"}},
	{"生活", []string{"搞笑", "日常", "美食圈", "动物圈", "手工", "绘画", "ASMR", "运动", "其他"}},
	{"鬼畜", []string{"鬼畜调教", "音MAD", "人力VOCALOID", "教程演示"}},
	{"时尚", []string{"美妆", "服饰", "健身", "资讯"}},
	{"广告", nil},
	{"娱乐", []string{"综艺", "明星", "Korea相关"}},
	{"影视", []string{"影视杂谈", "影视剪辑", "短片", "预告·资讯", "特摄"}},
	{"放映厅", []string{"纪录片", "电影", "电视剧"}},
}

func initVideoZoneAllowed() map[string]struct{} {
	out := make(map[string]struct{})
	for _, cat := range videoZoneCatalog {
		out[cat.parent] = struct{}{}
		for _, sub := range cat.subs {
			out[cat.parent+"-"+sub] = struct{}{}
		}
	}
	return out
}
