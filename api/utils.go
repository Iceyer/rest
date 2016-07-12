package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"pkg.deepin.io/server/utils/config"
	"pkg.deepin.io/server/utils/db"
	. "pkg.deepin.io/server/utils/logger"
)

var (
	ErrInvaildID  = errors.New("Invaild ID")
	ErrBodyFormat = errors.New("Invaild Body Content Format")
)

func mysqlEscape(str string) (ret string) {
	return strings.Replace(str, "_", "\\_", -1)
}

//Page Query will check
func PageQuery(c *gin.Context, v interface{}) error {
	// TODO: However check v must be *[]
	c.Request.ParseForm()
	page, _ := strconv.Atoi(c.Request.Form.Get("page"))
	c.Request.Form.Del("page")
	if page < 1 {
		page = 1
	}
	count, _ := strconv.Atoi(c.Request.Form.Get("count"))
	c.Request.Form.Del("count")
	if count < 1 {
		count = 20
	}
	if count > 2000 {
		//Return Error
		return errors.New("InvaildCount")
	}
	s := (page - 1) * count

	order := c.Request.Form.Get("order")
	switch order {
	case "asc":
	case "desc":
	default:
		order = "asc"
	}

	mode := c.Request.Form.Get("mode")
	precise := false
	if "precise" == mode {
		precise = true
	}

	t := reflect.ValueOf(v).Elem().Type().Elem()
	tv := reflect.New(reflect.ValueOf(v).Elem().Type())
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		tv = tv.Elem()
	}

	pri_key := "id"
	mdb := db.Maria
	cdb := mdb.Model(tv.Interface())
	//Check Json Tag First
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Tag.Get("gorm") == "primary_key" {
			pri_key = gorm.ToDBName(f.Name)
		}
		jsonKey := f.Tag.Get("json")
		if "" == jsonKey {
			jsonKey = strings.ToLower(f.Name)
		}
		queryValues, ok := c.Request.Form[jsonKey]
		if !ok {
			continue
		}

		inValues := [](interface{}){}
		inValuesStr := []int{}
		inExprision := " "
		for _, f := range queryValues {
			intv, _ := strconv.Atoi(f)
			inValuesStr = append(inValuesStr, intv)
			inValues = append(inValues, f)
			inExprision += "?,"
		}
		inExprision = inExprision[0 : len(inExprision)-1]

		k := f.Tag.Get("sqlname")
		if "" == k {
			k = f.Name
		}
		k = gorm.ToDBName(k)

		if f.Type.Kind() == reflect.Slice {
			sort.Ints(inValuesStr)
			valdata, _ := json.Marshal(inValuesStr)
			valstr := string(valdata)
			//valstr = strings.Replace(valstr, "[", "", -1)
			//valstr = strings.Replace(valstr, "]", "", -1)
			inFormat := fmt.Sprintf("`%v` = ?", k)
			mdb = mdb.Where(inFormat, valstr)
			cdb = cdb.Where(inFormat, valstr)
		} else {
			if 1 == len(inValues) {
				//for int/bool
				inFormat := fmt.Sprintf("`%v` = ?", k)
				switch f.Type.Kind() {
				case reflect.String:
					inFormat := fmt.Sprintf("`%v` like ?", k)
					if precise {
						inFormat = fmt.Sprintf("`%v` in ( %v )", k, "?")
						mdb = mdb.Where(inFormat, inValues[0])
						cdb = cdb.Where(inFormat, inValues[0])
					} else {
						mdb = mdb.Where(inFormat, mysqlEscape(fmt.Sprintf("%%%v%%", inValues[0])))
						cdb = cdb.Where(inFormat, mysqlEscape(fmt.Sprintf("%%%v%%", inValues[0])))
					}
				case reflect.Bool:
					vb := true
					if "false" == fmt.Sprint(inValues[0]) {
						vb = false
					}
					inFormat := fmt.Sprintf("`%v` = ?", k)
					mdb = mdb.Where(inFormat, vb)
					cdb = cdb.Where(inFormat, vb)
				case reflect.Int:
					id, err := strconv.Atoi(fmt.Sprint(inValues[0]))
					if nil != err {
						return err
					}
					mdb = mdb.Where(inFormat, id)
					cdb = cdb.Where(inFormat, id)
				default:
					inFormat := fmt.Sprintf("`%v` = ?", k)
					mdb = mdb.Where(inFormat, fmt.Sprintf("%v", inValues[0]))
					cdb = cdb.Where(inFormat, fmt.Sprintf("%v", inValues[0]))
				}
			} else {
				switch f.Type.Kind() {
				case reflect.Slice:
					Logger.Info("Should Not Here")
				default:
					inFormat := fmt.Sprintf("`%v` in ( %v )", k, inExprision)
					mdb = mdb.Where(inFormat, inValues...)
					cdb = cdb.Where(inFormat, inValues...)
				}
			}
		}
	}

	total := 0
	cdb.Count(&total)
	if err := mdb.Limit(count).Offset(s).Order(pri_key + " " + order).Find(v).Error; nil != err {
		return err
	}

	last := total / count
	if total%count > 0 {
		last += 1
	}
	// Set Pageable Header
	c.Writer.Header().Set("X-Total-Resource", fmt.Sprint(total))
	c.Writer.Header().Set("X-Total-Page", fmt.Sprint(last))
	c.Writer.Header().Set("X-Current-Page", fmt.Sprint(page))

	// Set Stand Link for pageable
	query := c.Request.URL.Query()
	query.Set("count", fmt.Sprint(count))
	query.Del("page")
	url := config.Read("Schema") + "://" + c.Request.Host + c.Request.URL.Path

	query.Set("page", fmt.Sprint(last))
	rawQuery := fmt.Sprintf(`<%s?%s>; rel="%s"`, url, query.Encode(), "last")
	c.Writer.Header().Set("Link", rawQuery)

	query.Set("page", fmt.Sprint(1))
	rawQuery = fmt.Sprintf(`<%s?%s>; rel="%s"`, url, query.Encode(), "first")
	c.Writer.Header().Add("Link", rawQuery)

	if page < last {
		query.Set("page", fmt.Sprint(page+1))
		rawQuery = fmt.Sprintf(`<%s?%s>; rel="%s"`, url, query.Encode(), "next")
		c.Writer.Header().Add("Link", rawQuery)
	}

	if page > 1 {
		query.Set("page", fmt.Sprint(page-1))
		rawQuery = fmt.Sprintf(`<%s?%s>; rel="%s"`, url, query.Encode(), "prev")
		c.Writer.Header().Add("Link", rawQuery)
	}

	if 0 == reflect.ValueOf(v).Elem().Len() {
		reflect.ValueOf(v).Elem().Set(reflect.MakeSlice(reflect.SliceOf(t), 0, 0))
	}
	return nil
}
