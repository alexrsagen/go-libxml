package libxml

import (
	"reflect"
	"strings"
)

type xmlTag struct {
	namespace     string
	tagName       string
	flagOmitEmpty bool
	flagAttribute bool
	flagChardata  bool
	flagComment   bool
	flagAny       bool
	flagInnerXML  bool
	flagCDATA     bool
}

func parseXMLTag(tag reflect.StructTag) *xmlTag {
	if tag == "" {
		return nil
	}
	specificTag := tag.Get("xml")
	if specificTag == "" {
		return nil
	}
	parsedTag := &xmlTag{}
	inFlag := false
	inNamespace := true
	inTagName := false
	flagBuf := ""
	for _, char := range specificTag {
		switch char {
		case ' ':
			if inNamespace {
				inNamespace = false
				inTagName = true
			}
		case ',':
			inNamespace = false
			inTagName = false
			if inFlag {
				switch strings.ToLower(flagBuf) {
				case "omitempty":
					parsedTag.flagOmitEmpty = true
				case "attr":
					parsedTag.flagAttribute = true
				case "chardata":
					parsedTag.flagChardata = true
				case "comment":
					parsedTag.flagComment = true
				case "any":
					parsedTag.flagAny = true
				case "innerxml":
					parsedTag.flagInnerXML = true
				case "cdata":
					parsedTag.flagCDATA = true
				}
			}
			inFlag = true
			flagBuf = ""
		default:
			if inNamespace {
				parsedTag.namespace += string(char)
			} else if inTagName {
				parsedTag.tagName += string(char)
			} else if inFlag {
				flagBuf += string(char)
			}
		}
	}
	if inFlag && flagBuf != "" {
		switch strings.ToLower(flagBuf) {
		case "omitempty":
			parsedTag.flagOmitEmpty = true
		case "attr":
			parsedTag.flagAttribute = true
		case "chardata":
			parsedTag.flagChardata = true
		case "comment":
			parsedTag.flagComment = true
		case "any":
			parsedTag.flagAny = true
		case "innerxml":
			parsedTag.flagInnerXML = true
		case "cdata":
			parsedTag.flagCDATA = true
		}
	}
	if parsedTag.tagName == "" && parsedTag.namespace != "" {
		parsedTag.tagName = parsedTag.namespace
		parsedTag.namespace = ""
	}
	return parsedTag
}

func stripTagPrefix(tag string) string {
	pos := strings.LastIndexByte(tag, ':')
	if pos == -1 {
		return tag
	}
	return tag[pos+1:]
}

func splitTagPrefix(tag string) (prefix, tagName string) {
	pos := strings.LastIndexByte(tag, ':')
	if pos == -1 {
		return "", tag
	}
	return tag[:pos], tag[pos+1:]
}
