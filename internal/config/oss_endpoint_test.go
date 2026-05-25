package config

import "testing"

func TestNormalizeAliyunOSSEndpoint(t *testing.T) {
	cases := []struct {
		endpoint, bucket, want string
	}{
		{
			"https://your-bucket.oss-cn-beijing.aliyuncs.com",
			"your-bucket",
			"https://oss-cn-beijing.aliyuncs.com",
		},
		{
			"http://MyBucket.oss-cn-hangzhou.aliyuncs.com",
			"mybucket",
			"http://oss-cn-hangzhou.aliyuncs.com",
		},
		{
			"https://oss-cn-beijing.aliyuncs.com",
			"your-bucket",
			"https://oss-cn-beijing.aliyuncs.com",
		},
		{
			"https://wrong.oss-cn-beijing.aliyuncs.com",
			"your-bucket",
			"https://wrong.oss-cn-beijing.aliyuncs.com",
		},
		{
			"https://your-bucket.oss-cn-beijing-internal.aliyuncs.com",
			"your-bucket",
			"https://oss-cn-beijing-internal.aliyuncs.com",
		},
	}
	for _, tc := range cases {
		if got := normalizeAliyunOSSEndpoint(tc.endpoint, tc.bucket); got != tc.want {
			t.Errorf("normalize(%q, %q) = %q, want %q", tc.endpoint, tc.bucket, got, tc.want)
		}
	}
}
