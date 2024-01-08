package openapi

import (
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/mutate"
	"github.com/yaklang/yaklang/common/openapi/openapi3"
	yaml "github.com/yaklang/yaklang/common/openapi/openapiyaml"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/yak/yaklib/codec"
	"strings"
)

func v3Generator(t string, config *OpenAPIConfig) error {
	var data openapi3.T
	jsonT, err := yaml.YAMLToJSON([]byte(t))
	if err == nil {
		t = string(jsonT)
	}
	err = data.UnmarshalJSON([]byte(t))
	if err != nil {
		return utils.Wrapf(err, "unmarshal openapi3 failed")
	}
	if config == nil {
		config = NewDefaultOpenAPIConfig()
	}

	var root mutate.FuzzHTTPRequestIf
	root, err = mutate.NewFuzzHTTPRequest(`GET / HTTP/1.1
Host: www.example.com
`)
	if err != nil {
		return utils.Wrapf(err, "create http request failed")
	}
	baseUrl, _ := data.Servers.BasePath()
	if baseUrl != "" {
		baseUrl = strings.TrimRight(baseUrl, "/")
		root = root.FuzzPath(baseUrl)
	}

	for _, pathStr := range data.Paths.InMatchingOrder() {
		pathIns := data.Paths.Value(pathStr)
		log.Infof("path: %v, ops: %v", pathStr, len(pathIns.Operations()))
		pathRoot := root.FuzzPathAppend(pathStr)
		for op, ins := range pathIns.Operations() {
			methodRoot := pathRoot.FuzzMethod(op)
			pr := methodRoot.FirstFuzzHTTPRequest().GetPath()
			var originPath, _ = codec.PathUnescape(pr)
			if originPath == "" {
				originPath = pr
			}

			for _, parameter := range ins.Parameters {
				param, err := v3_parameterToValue(data, parameter)
				if err != nil {
					log.Errorf("v3_parameterToValue [%v] failed: %v", param.Name, err)
					continue
				}
				scheme, err := v3_schemaToValue(data, param.Schema)
				if err != nil {
					log.Errorf("v3_schemaToValue [%v] failed: %v", param.Name, err)
					continue
				}
				switch param.In {
				case openapi3.ParameterInQuery:
					methodRoot = methodRoot.FuzzGetParams(param.Name, ValueViaField(param.Name, scheme.Type))
				case openapi3.ParameterInHeader:
					methodRoot = methodRoot.FuzzHTTPHeader(param.Name, ValueViaField(param.Name, scheme.Type))
				case openapi3.ParameterInPath:
					methodRoot = methodRoot.FuzzPath
				case openapi3.ParameterInCookie:
					methodRoot = methodRoot.FuzzCookie(param.Name, ValueViaField(param.Name, scheme.Type))
				}
			}
		}
	}
	return nil
}
