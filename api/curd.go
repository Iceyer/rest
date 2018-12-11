package api

import (
	"github.com/gin-gonic/gin"
	"pkg.deepin.io/server/utils/logger"
)

type CURD interface {
	GetBy(key string, value ...interface{}) error
	Create() error
	Check() error
	Delete() error
	Data() interface{}
}

func Create(curd CURD) gin.HandlerFunc {
	return func(c *gin.Context) {
		rr := NewRsp(c)
		defer rr.Render()

		if err := curd.Check(); nil != err {
			logger.Warning("Check object %v failed: %v", curd.Data(), err)
			rr.Error(400, NewError(err, ""))
			return
		}

		if err := curd.Create(); nil != err {
			logger.Error("Create object %v failed: %v", curd.Data(), err)
			rr.Error(400, NewError(err, ""))
			return
		}
		rr.Data = curd.Data()
	}
}

func Delete(curd CURD) gin.HandlerFunc {
	return func(c *gin.Context) {
		rr := NewRsp(c)
		defer rr.Render()

		if err := curd.Delete(); nil != err {
			logger.Error("%v", err)
			rr.Error(400, NewError(err, ""))
			return
		}

		v, _ := c.Get("Callback")
		if callback, _ := v.(func(interface{})); callback != nil {
			callback(curd)
		}
		rr.Data = curd.Data()
	}
}
