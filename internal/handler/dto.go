package handler

// okResponse is the standard success body for endpoints returning only {"ok": true}.
type okResponse struct {
	OK bool `json:"ok"`
}

// deletedResponse is the success body for delete endpoints returning {"deleted": true}.
type deletedResponse struct {
	Deleted bool `json:"deleted"`
}

// imageURLResponse is the success body for upload endpoints returning {"image_url": ...}.
type imageURLResponse struct {
	ImageURL string `json:"image_url"`
}
