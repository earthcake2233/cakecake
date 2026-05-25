package userlevel

// MaxLevel is the highest user level (Lv6).
const MaxLevel = 6

// Thresholds[i] is the minimum total experience required to reach level i+1.
// Lv1: 0, Lv2: 20, Lv3: 150, Lv4: 450, Lv5: 1080, Lv6: 2880.
var Thresholds = []uint64{0, 20, 150, 450, 1080, 2880}

// Info matches the Bilibili-style level_info object used by the frontend.
type Info struct {
	CurrentLevel int    `json:"current_level"`
	CurrentMin   uint64 `json:"current_min"`
	CurrentExp   uint64 `json:"current_exp"`
	NextExp      uint64 `json:"next_exp"`
}

// FromExperience derives level_info from total experience points.
func FromExperience(exp uint64) Info {
	lv := 1
	for i := MaxLevel - 1; i >= 0; i-- {
		if exp >= Thresholds[i] {
			lv = i + 1
			break
		}
	}
	curMin := Thresholds[lv-1]
	var nextExp uint64
	if lv >= MaxLevel {
		// 满级后展示为 当前经验/2880（Lv6 门槛），进度条视为已满。
		nextExp = Thresholds[MaxLevel-1]
	} else {
		nextExp = Thresholds[lv]
	}
	return Info{
		CurrentLevel: lv,
		CurrentMin:   curMin,
		CurrentExp:   exp,
		NextExp:      nextExp,
	}
}
