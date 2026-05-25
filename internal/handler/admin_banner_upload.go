package handler

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/model"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/resp"
)

func (a *API) uploadBannerImageToOSS(fh *multipart.FileHeader, objectKey string) (string, int) {
	if fh == nil {
		return "", errcode.CodeParamError
	}
	if code := coverval.ValidateCoverHeader(fh); code != 0 {
		return "", code
	}
	if a.OSS == nil {
		return "", errcode.CodeInternalError
	}
	if err := os.MkdirAll(a.Cfg.TempUploadDir, 0o755); err != nil {
		return "", errcode.CodeInternalError
	}
	tmp := filepath.Join(a.Cfg.TempUploadDir, uuid.NewString()+filepath.Ext(fh.Filename))
	if err := saveUploadedFile(fh, tmp); err != nil {
		return "", errcode.CodeInternalError
	}
	defer os.Remove(tmp)
	if err := a.OSS.UploadFile(objectKey, tmp); err != nil {
		a.Log.Error("oss banner image upload", zap.String("key", objectKey), zap.Error(err))
		return "", errcode.CodeInternalError
	}
	return a.Cfg.OSSObjectURL(objectKey), 0
}

func bannerImageExt(fh *multipart.FileHeader) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	if ext == "jpeg" {
		ext = "jpg"
	}
	if ext == "" {
		ext = "jpg"
	}
	return ext
}

// AdminUploadBannerImage POST /api/v1/admin/home-banners/upload-image — multipart field "image".
func (a *API) AdminUploadBannerImage(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(12 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	fh, err := c.FormFile("image")
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	key := fmt.Sprintf("home-banners/%s.%s", uuid.NewString(), bannerImageExt(fh))
	url, code := a.uploadBannerImageToOSS(fh, key)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	resp.OK(c, gin.H{"image_url": url})
}

// AdminUploadBannerImageByID POST /api/v1/admin/home-banners/:id/image — replace slide image on OSS.
func (a *API) AdminUploadBannerImageByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var b model.HomeBanner
	if err := a.DB.First(&b, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := c.Request.ParseMultipartForm(12 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	fh, err := c.FormFile("image")
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	oldURL := b.ImageURL
	key := fmt.Sprintf("home-banners/%d.%s", b.ID, bannerImageExt(fh))
	url, code := a.uploadBannerImageToOSS(fh, key)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	if err := a.DB.Model(&b).Update("image_url", url).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if oldURL != "" && oldURL != url {
		purgeBannerImageURL(a.Cfg, a.OSS, a.Log, oldURL)
	}
	resp.OK(c, gin.H{"image_url": url})
}
