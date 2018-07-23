package libxml

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/jbowtie/gokogiri/xml"
)

const nsCleanParseOption = xml.XML_PARSE_RECOVER |
	xml.XML_PARSE_NONET |
	xml.XML_PARSE_NOERROR |
	xml.XML_PARSE_NOWARNING |
	xml.XML_PARSE_NSCLEAN

var ErrNoRoot = errors.New("XMLName not defined on root element")

// Marshal returns the XML encoding of v.¨
//
// See https://golang.org/pkg/encoding/xml/#Marshal for more information.
func Marshal(v interface{}) (string, error) {
	// Attempt to resolve struct/structpointer
	if v == nil {
		return "", ErrInvalidStructPtr
	}
	s := reflect.ValueOf(v)
	if s.Type().Kind() == reflect.Ptr {
		s = s.Elem()
	}
	if s.NumField() == 0 {
		return "", nil
	}
	t := s.Type()

	// Create document
	doc := xml.CreateEmptyDocument([]byte("UTF-8"), []byte("UTF-8"))

	// Get root node tag name and namespace
	xmlNameField, ok := t.FieldByName("XMLName")
	if !ok {
		return "", ErrNoRoot
	}
	xmlNameTag := parseXMLTag(xmlNameField.Tag)

	// Create root node
	root := doc.CreateElementNode(xmlNameTag.tagName)
	if xmlNameTag.namespace != "" {
		root.SetNamespace("", xmlNameTag.namespace)
	}
	doc.AddChild(root)

	// Fill values
	err := fillNodeFromStruct(doc, root, s)
	if err != nil {
		return "", err
	}

	// Clean up namespaces
	cleanDoc, err := xml.Parse([]byte(doc.String()), []byte("UTF-8"), []byte{}, nsCleanParseOption, []byte("UTF-8"))
	if err != nil {
		return "", err
	}

	return cleanDoc.String(), nil
}

func getStringFromVal(t reflect.Type, s reflect.Value) (string, error) {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(s.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(s.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(s.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(s.Float(), 'f', -1, 64), nil
	case reflect.String:
		return s.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(s.Bool()), nil
	}
	return "", fmt.Errorf("Type %s (%s) not implemented", t.Kind().String(), t.String())
}

func setContentFromStructField(doc *xml.XmlDocument, t reflect.Type, s reflect.Value, n xml.Node) error {
	switch t.Kind() {
	case reflect.Ptr:
		return setContentFromStructField(doc, t.Elem(), s.Elem(), n)
	case reflect.Interface:
		if reflect.DeepEqual(s.Interface(), reflect.Zero(t).Interface()) {
			return nil
		}
		return setContentFromStructField(doc, s.Elem().Type(), s.Elem(), n)
	case reflect.Struct:
		return fillNodeFromStruct(doc, n, s)
	default:
		str, err := getStringFromVal(t, s)
		if err != nil {
			return err
		}
		n.SetContent(str)
	}
	return nil
}

func setAttrFromStructField(t reflect.Type, s reflect.Value, n xml.Node, attrName string) error {
	switch t.Kind() {
	case reflect.Ptr:
		return setAttrFromStructField(t.Elem(), s.Elem(), n, attrName)
	default:
		str, err := getStringFromVal(t, s)
		if err != nil {
			return err
		}
		n.SetAttr(attrName, str)
	}
	return nil
}

func fillNodeFromStruct(doc *xml.XmlDocument, n xml.Node, s reflect.Value) error {
	t := s.Type()

	for i := 0; i < s.NumField(); i++ {
		tField := t.Field(i)
		if tField.Name == "XMLName" {
			continue
		}
		sField := s.Field(i)

		// Get XML tag name and namespace from struct definition
		tagNS := ""
		fullTagName := ""
		isAttr := false
		if xmlTag := parseXMLTag(tField.Tag); xmlTag != nil {
			if reflect.DeepEqual(sField.Interface(), reflect.Zero(tField.Type).Interface()) && xmlTag.flagOmitEmpty {
				continue
			} else if xmlTag.flagAttribute {
				isAttr = true
			} else if xmlTag.flagChardata {
				err := setContentFromStructField(doc, tField.Type, sField, n)
				if err != nil {
					return err
				}
				continue
			}
			fullTagName = xmlTag.tagName
			tagNS = xmlTag.namespace
		} else {
			// Fall back to field name if tag name is empty
			fullTagName = tField.Name
		}

		// Fall back to type name if tag name is empty
		if fullTagName == "" {
			if tField.Type.Kind() == reflect.Ptr {
				fullTagName = sField.Elem().Type().String()
			} else {
				fullTagName = tField.Type.String()
			}
		}

		prefix, tagName := splitTagPrefix(fullTagName)

		if isAttr {
			err := setAttrFromStructField(tField.Type, sField, n, tagName)
			if err != nil {
				return err
			}
		} else {
			if tField.Type.Kind() == reflect.Slice {
				for i := 0; i < sField.Len(); i++ {
					node := doc.CreateElementNode(tagName)
					if tagNS != "" {
						node.SetNamespace(prefix, tagNS)
					}
					err := setContentFromStructField(doc, sField.Type().Elem(), sField.Index(i), node)
					if err != nil {
						return err
					}
					n.AddChild(node)
				}
			} else {
				node := doc.CreateElementNode(tagName)
				if tagNS != "" {
					node.SetNamespace(prefix, tagNS)
				}
				err := setContentFromStructField(doc, tField.Type, sField, node)
				if err != nil {
					return err
				}
				n.AddChild(node)
			}
		}
	}

	return nil
}
