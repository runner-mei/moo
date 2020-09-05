package services

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/users/welcome"
)

const UMEXTENFFIELDS = "um_extend_fields.json"

var (
	WhiteAddressList = Field{ID: "white_address_list",
		Name:      "登录IP",
		IsDefault: "true",
		Type:      "text"}
	WelcomeURL = Field{ID: welcome.FieldName,
		Name:      "首页",
		IsDefault: "true",
		Type:      "text"}
	Email = Field{ID: "email",
		Name:      "邮箱",
		IsDefault: "true",
		Type:      "text"}
	Phone = Field{ID: "phone",
		Name:      "电话",
		IsDefault: "true",
		Type:      "text"}

	DefaultFields = []Field{
		WhiteAddressList,
		WelcomeURL,
		Phone,
		Email,
	}
)

type Field struct {
	Category     string `json:"category,omitempty"`
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Type         string `json:"type,omitempty"`
	DefaultValue string `json:"default,omitempty"`
	IsDefault    string `json:"-"`
	Editor       string `json:"editor,omitempty"`

	Enumerations []welcome.InputOption `json:"enumerations,omitempty"`
}

type Fields struct {
	Title    string  `json:"title"`
	Prefix   string  `json:"prefix"`
	Sequence int     `json:"sequence"`
	Fields   []Field `json:"fields"`
}

func ReadFieldsFromDir(env *moo.Environment) ([]Fields, error) {
	defaultFields, err := ReadFieldsFromFile(env, DefaultFields)
	if err != nil {
		return nil, err
	}

	results := make([]Fields, 0, 3+1)
	results = append(results, Fields{Title: "", Fields: defaultFields})

	filePattern := env.Fs.FromConfig("/**/user_fields*.json")
	files, err := filepath.Glob(filePattern)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(errors.Wrap(err, "ReadFieldsFromDir: read '"+filePattern+"' fail").Error())
		}
		return results, nil
	}

	for _, filename := range files {
		var fields Fields
		err := util.FromHjsonFile(filename, &fields)
		if err != nil {
			log.Println(errors.Wrap(err, "ReadFieldsFromDir: read '"+filename+"' fail"))
			continue
		}

		if fields.Prefix != "" && !strings.HasSuffix(fields.Prefix, ".") {
			fields.Prefix = fields.Prefix + "."
		}

		for idx := range fields.Fields {
			isEmpty := true
			for eidx := range fields.Fields[idx].Enumerations {
				if fields.Fields[idx].Enumerations[eidx].Label != "" {
					isEmpty = false
					break
				}
			}

			if isEmpty {
				for eidx := range fields.Fields[idx].Enumerations {
					fields.Fields[idx].Enumerations[eidx].Label = fields.Fields[idx].Enumerations[eidx].Value
				}
			}
		}

		results = append(results, fields)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Sequence < results[j].Sequence
	})
	return results, nil
}

func ReadFieldsFromFile(env *moo.Environment, defaultFields []Field) ([]Field, error) {
	filename := env.Fs.FromDataConfig(UMEXTENFFIELDS)

	var fields []Field
	err := util.FromHjsonFile(filename, &fields)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultFields, nil
		}
		return nil, errors.New("read '" + filename + "' fail: " + err.Error())
	}

	results := make([]Field, 0, len(defaultFields)+len(fields))
	results = append(results, defaultFields...)
	for _, field := range fields {
		foundIdx := -1
		for idx := range defaultFields {
			if defaultFields[idx].ID == field.ID {
				foundIdx = idx
				break
			}
		}

		isEmpty := true
		for eidx := range field.Enumerations {
			if field.Enumerations[eidx].Label != "" {
				isEmpty = false
				break
			}
		}

		if isEmpty {
			for eidx := range field.Enumerations {
				field.Enumerations[eidx].Label = field.Enumerations[eidx].Value
			}
		}

		if foundIdx < 0 {
			results = append(results, field)
		} else {
			results[foundIdx] = field
		}

	}
	return results, nil
}
