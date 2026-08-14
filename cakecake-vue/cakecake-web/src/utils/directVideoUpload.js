/** 直传 OSS：用后端签发的 presigned URL 把文件直接 PUT 到对象存储，
 *  不经过 API 服务器带宽。onProgress(loaded, total) 可选；成功时 resolve。
 *  contentType 必须与签发 ticket 时提交的类型一致（OSS 把它算进签名），
 *  默认取 file.type，空则用 application/octet-stream。 */
export function uploadToPresignedURL(url, file, onProgress, contentType) {
  const ct =
    (contentType && String(contentType).trim()) ||
    (file && file.type) ||
    "application/octet-stream";
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", url);
    xhr.setRequestHeader("Content-Type", ct);
    xhr.upload.onprogress = e => {
      if (onProgress && e.lengthComputable && e.total > 0) {
        onProgress(e.loaded, e.total);
      }
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
      } else if (xhr.status === 0) {
        reject(
          new Error(
            "上传失败：网络或服务配置异常，请稍后重试"
          )
        );
      } else {
        reject(new Error(`上传失败：服务返回异常（HTTP ${xhr.status}），请稍后重试`));
      }
    };
    xhr.onerror = () =>
      reject(
        new Error(
          "上传失败：网络异常，请检查网络后重试"
        )
      );
    xhr.send(file);
  });
}
