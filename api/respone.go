package api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
)

var (
	ErrInvildInput       = fmt.Errorf("Error Input Format")
	ErrIllegalDataFormat = fmt.Errorf("illegal_data_format")
	InterError           = fmt.Errorf("Interal Error")
)

type Error interface {
	Error() error
	Description() interface{}
}

type apiError struct {
	err         error
	description interface{}
}

func (ae *apiError) Error() error {
	return ae.err
}

func (ae *apiError) Description() interface{} {
	return ae.description
}

func NewError(err error, des interface{}) *apiError {
	return &apiError{
		err:         err,
		description: des,
	}
}

type Rsp struct {
	Data interface{} `json:"-"`

	ErrorCode        int         `json:"-"`
	ErrorMsg         string      `json:"message"`
	ErrorDescription interface{} `json:"description"`
	isError          bool
	isRaw            bool

	c *gin.Context
}

func NewRsp(c *gin.Context) *Rsp {
	return &Rsp{
		c:       c,
		isError: false,
	}
}

func (r *Rsp) Render() {
	if r.isError {
		r.c.JSON(r.ErrorCode, r)
	} else {
		//Support jsonp
		jsonp := r.c.Request.Form.Get("jsonp")
		jdata, err := json.Marshal(r.Data)
		if nil != err {
			r.Error(500, NewError(InterError, err))
			r.c.JSON(r.ErrorCode, r)
		}
		fmt.Println("test", jsonp, r.Data)
		if "" != jsonp {
			r.c.Writer.WriteHeader(200)
			r.c.Writer.WriteString(fmt.Sprintf("%s(%s)", jsonp, string(jdata)))
		} else {
			r.c.JSON(200, r.Data)
		}
	}
}

func (rsp *Rsp) WriteRaw(data interface{}) {
	rsp.isRaw = true
	rsp.Data = data
}

func (rsp *Rsp) SetError(code int, err error) {
	rsp.isError = true
	if InterError == err {
		code = 500
	}
	rsp.ErrorCode = code
	rsp.ErrorMsg = fmt.Sprint(err)
}

func (rsp *Rsp) Error(code int, err Error) {
	rsp.isError = true
	if InterError == err.Error() {
		code = 500
	}
	rsp.ErrorCode = code
	rsp.ErrorMsg = err.Error().Error()
	rsp.ErrorDescription = err.Description()
}
