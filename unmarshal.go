package libxml

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	goxml "encoding/xml"

	"github.com/jbowtie/gokogiri"
	"github.com/jbowtie/gokogiri/xml"
)

// ErrInvalidStructPtr is returned when the v argument of Unmarshal is not a valid pointer to a struct
var ErrInvalidStructPtr = errors.New("Not a valid struct pointer")

var xmlNameType = reflect.TypeOf(goxml.Name{})

// Unmarshal parses the XML-encoded data and stores the result in the value pointed to by v, which must be an
// arbitrary struct, slice, or string. Well-formed data that does not fit into v is discarded.
//
// See https://golang.org/pkg/encoding/xml/#Unmarshal for more information.
func Unmarshal(data string, v interface{}) error {
	// Attempt to resolve to a struct pointer
	if v == nil {
		return ErrInvalidStructPtr
	}
	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		return ErrInvalidStructPtr
	}
	s := reflect.ValueOf(v).Elem()
	if s.NumField() == 0 {
		return nil
	}

	// Parse XML document with libxml2 binding
	doc, err := gokogiri.ParseXml([]byte(data))
	if err != nil {
		return err
	}
	defer doc.Free()

	// Get document root and fill s with values from elements under root
	err = fillStructFromNode(doc.Root(), s)
	if err != nil {
		return err
	}

	return nil
}

func setValFromString(t reflect.Type, s reflect.Value, str string) error {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(str, 0, 64)
		if err != nil {
			return err
		}
		if s.OverflowInt(val) {
			return errors.New("Int value too big: " + str)
		}
		s.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(str, 0, 64)
		if err != nil {
			return err
		}
		if s.OverflowUint(val) {
			return errors.New("UInt value too big: " + str)
		}
		s.SetUint(val)
	case reflect.Float32:
		val, err := strconv.ParseFloat(str, 32)
		if err != nil {
			return err
		}
		s.SetFloat(val)
	case reflect.Float64:
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		s.SetFloat(val)
	case reflect.String:
		s.SetString(str)
	case reflect.Bool:
		val, err := strconv.ParseBool(str)
		if err != nil {
			return err
		}
		s.SetBool(val)
	}
	return fmt.Errorf("Type %s (%s) not implemented", t.String(), t.Kind().String())
}

func setStructFieldFromContent(t reflect.Type, s reflect.Value, n xml.Node) error {
	switch t.Kind() {
	case reflect.Ptr:
		sType := t.Elem()
		if s.IsNil() {
			val := reflect.New(sType)
			err := setStructFieldFromContent(sType, val.Elem(), n)
			s.Set(val)
			return err
		}
		return setStructFieldFromContent(sType, s.Elem(), n)
	case reflect.Interface:
		if s.IsNil() {
			return nil
		}
		sType := s.Elem().Type()
		val := reflect.New(sType)
		err := setStructFieldFromContent(sType, val.Elem(), n)
		s.Set(val)
		return err
	case reflect.Struct:
		return fillStructFromNode(n, s)
	}
	return setValFromString(t, s, n.Content())
}

func setStructFieldFromAttr(t reflect.Type, s reflect.Value, n xml.Node, attrName string) error {
	attr := n.Attribute(attrName)
	if attr == nil {
		return errors.New("No attribute defined")
	}
	switch t.Kind() {
	case reflect.Ptr:
		sType := t.Elem()
		val := reflect.New(sType)
		err := setStructFieldFromAttr(sType, val, n, attrName)
		s.Set(val)
		return err
	}
	return setValFromString(t, s, attr.Content())
}

func fillStructFromNode(n xml.Node, s reflect.Value) error {
	t := s.Type()

	// Loop over struct fields
	for i := 0; i < s.NumField(); i++ {
		tField := t.Field(i)
		sField := s.Field(i)
		if !sField.CanSet() && tField.Type.Kind() != reflect.Struct {
			continue
		}

		// Set XMLName field
		if tField.Name == "XMLName" && tField.Type == xmlNameType {
			sField.Set(reflect.ValueOf(goxml.Name{
				Space: n.Namespace(),
				Local: n.Name(),
			}))
		}

		// Get XML tag name and namespace from struct definition
		tagNS := ""
		tagName := ""
		if xmlTag := parseXMLTag(tField.Tag); xmlTag != nil {
			// Handle special struct tag flags
			if xmlTag.flagInnerXML {
				if tField.Type.Kind() == reflect.String {
					sField.SetString(n.String())
				} else if tField.Type.Kind() == reflect.Slice && tField.Type.Elem().Kind() == reflect.Uint8 {
					sField.SetBytes([]byte(n.String()))
				}
				continue
			} else if xmlTag.flagAttribute {
				setStructFieldFromAttr(tField.Type, sField, n, xmlTag.tagName)
				continue
			} else if xmlTag.flagChardata {
				if tField.Type.Kind() == reflect.String {
					sField.SetString(n.Content())
				} else if tField.Type.Kind() == reflect.Slice && tField.Type.Elem().Kind() == reflect.Uint8 {
					sField.SetBytes([]byte(n.Content()))
				}
				continue
			}
			tagName = xmlTag.tagName
			tagNS = xmlTag.namespace
		} else {
			// Fall back to field name if tag name is empty
			tagName = tField.Name
		}

		// Fall back to type name if tag name is empty
		if tagName == "" {
			if tField.Type.Kind() == reflect.Ptr {
				tagName = sField.Elem().Type().String()
			} else {
				tagName = tField.Type.String()
			}
		}

		// Strip namespace prefix from tag name
		tagName = stripTagPrefix(tagName)

		// Loop over child elements to find value(s) for current struct field
		docf := xml.DocumentFragment{Node: n}
		for _, c := range docf.Children() {
			if c.Name() == tagName && (tagNS == "" || c.Namespace() == tagNS) {
				if tField.Type.Kind() == reflect.Slice {
					// Handle slice fields
					elemType := tField.Type.Elem()
					isPtr := false
					if elemType.Kind() == reflect.Ptr {
						isPtr = true
						elemType = tField.Type.Elem().Elem()
					}
					elem := reflect.New(elemType)
					if isPtr {
						setStructFieldFromContent(elemType, elem.Elem(), c)
						sField.Set(reflect.Append(sField, elem))
					} else {
						setStructFieldFromContent(elemType, elem.Elem(), c)
						sField.Set(reflect.Append(sField, elem.Elem()))
					}
				} else {
					// Handle normal fields
					setStructFieldFromContent(tField.Type, sField, c)
				}
			}
		}
	}

	return nil
}
