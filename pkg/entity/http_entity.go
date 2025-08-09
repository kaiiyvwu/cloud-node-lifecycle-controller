package entity

// HTTPResponse http response struct
type HTTPResponse struct {
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
	Code    int32       `json:"code"`
	Message string      `json:"message"`
}

// Succ fast success method
func (resp *HTTPResponse) Succ() HTTPResponse {
	return resp.SuccessWithData(nil)
}

// SuccessWithData success with data method
func (resp *HTTPResponse) SuccessWithData(data interface{}) HTTPResponse {
	resp.Success = true
	resp.Code = 200
	resp.Data = data
	resp.Message = "success"
	return *resp
}
