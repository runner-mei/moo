package welcome

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"strings"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/resty"
)

const FieldName = "welcome_url"

type Config struct {
	Name        string `json:"name"`
	ListURL     string `json:"list_url"`
	RedirectURL string `json:"redirect_url"`
}

func ReadConfigs(env *moo.Environment) ([]Config, error) {
	filename := env.Fs.FromConfig("home.json")
	args := map[string]interface{}{
		"urlRoot": env.DaemonUrlPath,
	}

	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.New("ReadHTTPConfigFromFile: " + err.Error())
	}

	t, err := template.New("default").Funcs(template.FuncMap{
		"join": urlutil.Join,
	}).Parse(string(bs))
	if err != nil {
		return nil, errors.New("parse url template in '" + filename + "' fail: " + err.Error())
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, args); err != nil {
		return nil, errors.New("generate url template in '" + filename + "' fail: " + err.Error())
	}
	if buf.Len() == 0 {
		return nil, errors.New("template result in '" + filename + "' is empty.")
	}

	var config struct {
		Applications []Config `json:"applications,omitempty"`
	}

	bs = buf.Bytes()
	err = json.NewDecoder(&buf).Decode(&config)
	if err != nil {
		return nil, errors.New("read '" + filename + "' fail: " + err.Error() + "\r\n" + string(bs))
	}
	return config.Applications, nil
}

// InputOption - Value pair used to define an option for select and redio input fields.
type InputOption struct {
	Value string `json:"value" xorm:"value"`
	Label string `json:"label" xorm:"label"`
}

type GroupedOption struct {
	Label    string        `json:"label,omitempty"`
	Children []InputOption `json:"children,omitempty"`
}

func concatOptionSet(allList, parts []GroupedOption) []GroupedOption {
	for pidx := range parts {
		foundIdx := -1
		for idx := range allList {
			if allList[idx].Label == parts[pidx].Label {
				foundIdx = idx
				break
			}
		}

		if foundIdx >= 0 {
			if len(parts[pidx].Children) > 0 {
				allList[foundIdx].Children = append(allList[foundIdx].Children, parts[pidx].Children...)
			}
		} else {
			allList = append(allList, parts[pidx])
		}
	}
	return allList
}

func ReadURLs(env *moo.Environment, rootURL string) ([]GroupedOption, error) {
	apps, err := ReadConfigs(env)
	if err != nil {
		return nil, err
	}

	choices1, err := ReadWelcomeChoices(env)
	if err != nil {
		return nil, err
	}

	choices2, err := GetURLs(env, rootURL, apps)
	if err != nil {
		return nil, err
	}
	return concatOptionSet(choices1, choices2), nil
}

func GetURLs(env *moo.Environment, rootURL string, apps []Config) ([]GroupedOption, error) {
	var errList []error
	var allChoices []GroupedOption

	for _, app := range apps {
		if app.ListURL == "" {
			continue
		}

		listURL := app.ListURL
		if s := strings.ToLower(listURL); !strings.HasPrefix(s, "http://") &&
			!strings.HasPrefix(s, "https://") {

			if strings.HasPrefix(listURL, "/") {
				if strings.HasSuffix(rootURL, "/") {
					listURL = rootURL + strings.TrimPrefix(listURL, "/")
				} else {
					listURL = rootURL + listURL
				}
			} else {
				if strings.HasSuffix(rootURL, "/") {
					listURL = rootURL + listURL
				} else {
					listURL = rootURL + "/" + listURL
				}
			}
		}
		var buf = bytes.NewBuffer(make([]byte, 0, 4*1024))
		var choices []GroupedOption
		err := resty.Get(listURL, resty.Unmarshal(&choices, buf))
		if err != nil {
			errList = append(errList, errors.Wrap(err, "read from "+app.Name+"and url is "+listURL))
		} else {
			for idx := range choices {
				for cidx := range choices[idx].Children {
					choices[idx].Children[cidx].Value = app.Name + "," + choices[idx].Children[cidx].Value
				}
			}
			allChoices = concatOptionSet(allChoices, choices)
		}
	}

	if len(errList) > 0 {
		return allChoices, errors.ErrArray(errList, "GetWelcomeURLs")
	}
	return allChoices, nil
}

func ReadWelcomeChoices(env *moo.Environment) ([]GroupedOption, error) {
	filename := env.Fs.FromConfig("home.json")
	args := map[string]interface{}{
		"urlRoot": env.DaemonUrlPath,
	}

	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.New("ReadWelcomeChoices: " + err.Error())
	}

	t, err := template.New("default").Funcs(template.FuncMap{
		"join": urlutil.Join,
	}).Parse(string(bs))
	if err != nil {
		return nil, errors.New("parse welcome choices template in '" + filename + "' fail: " + err.Error())
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, args); err != nil {
		return nil, errors.New("generate welcome choices in '" + filename + "' fail: " + err.Error())
	}

	var config struct {
		Choices []GroupedOption `json:"choices,omitempty"`
	}

	bs = buf.Bytes()
	err = json.NewDecoder(&buf).Decode(&config)
	if err != nil {
		return nil, errors.New("read '" + filename + "' fail: " + err.Error() + "\r\n" + string(bs))
	}
	return config.Choices, nil
}
