package http

import (
	"reflect"

	"github.com/mit-dci/opencbdc-tctl/common"
)

type frontendTestRunField struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

func (h *HttpServer) testRunFieldList() []frontendTestRunField {
	var res []frontendTestRunField
	t := reflect.TypeOf(common.TestRun{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = field.Name
		}

		fieldTitle := field.Tag.Get("feFieldTitle")
		if fieldTitle == "" {
			continue
		}

		fieldType := field.Tag.Get("feFieldType")
		if fieldType == "" {
			continue
		}

		res = append(res, frontendTestRunField{
			Name:  fieldName,
			Title: fieldTitle,
			Type:  fieldType,
		})
	}
	return res
}
