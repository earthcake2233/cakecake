/** 直传 OSS：用后端签发的 presigned URL 把文件直接 PUT 到对象存储，
 *  不经过 API 服务器带宽。返回 Promise，成功时 resolve。 */
export function uploadToPresignedURL(url, file) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", url);
    xhr.upload.onprogress = () => {
      // 进度回调暂不展示；如需进度条，可在这里透出 loaded/total。
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
      } else {
        reject(new Error(`直传失败：HTTP ${xhr.status}`));
      }
    };
    xhr.onerror = () => reject(new Error("直传网络错误，请重试"));
    xhr.send(file);
  });
}
