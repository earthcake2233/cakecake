package config

import "strings"

// normalizeAliyunOSSEndpoint fixes a common misconfiguration: using virtual-hosted
// style (https://BUCKET.oss-region.aliyuncs.com) as OSS_ENDPOINT. The Aliyun Go
// SDK expects the regional endpoint only (https://oss-region.aliyuncs.com); otherwise
// PutObject targets BUCKET.BUCKET.oss-region... and TLS fails (wrong certificate SAN).
func normalizeAliyunOSSEndpoint(endpoint, bucket string) string {
	endpoint = strings.TrimSpace(endpoint)
	bucket = strings.TrimSpace(bucket)
	if endpoint == "" || bucket == "" {
		return endpoint
	}
	host := strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	host = strings.TrimSuffix(host, "/")
	if host == "" {
		return endpoint
	}
	if len(host) <= len(bucket)+1 || host[len(bucket)] != '.' {
		return endpoint
	}
	if !strings.EqualFold(host[:len(bucket)], bucket) {
		return endpoint
	}
	rest := host[len(bucket)+1:]
	if len(rest) < 4 || !strings.HasPrefix(strings.ToLower(rest), "oss-") {
		return endpoint
	}
	scheme := "https://"
	if strings.HasPrefix(strings.ToLower(endpoint), "http://") {
		scheme = "http://"
	}
	return scheme + rest
}
