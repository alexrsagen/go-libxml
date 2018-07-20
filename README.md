# go-libxml
An abstraction of a libxml2 binding which exports Marshal/Unmarshal like Golang encoding/xml API

As of today there are [13 open issues regarding XML](https://github.com/golang/go/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue+milestone%3AGo1.12+in%3Atitle+encoding%2Fxml+) in [the Go repository](https://github.com/golang/go/). This library aims to serve as a workaround for all the issues with the Go <1.12 encoding/xml implementation, as the language has delayed the fixes for several releases now.

Note that this library **is a workaround** and should be tested in all possible use cases before being used in production.

Thanks to [moovweb](https://github.com/moovweb/gokogiri) for the libxml wrapper, and thanks to [jbowtie](https://github.com/jbowtie/gokogiri) for the Go 1.9+ build fixes.

## Features
- Exact same API as the encoding/xml package, therefore easy to switch back once XML in Go is fixed
- Namespace cleanup on Marshal using libxml2 NSCLEAN flag

## Known bugs/missing features
- Marshal/Unmarshal of array types (fixed-length slices)
- Element nesting using `>` operator. Example struct tag that would fail: `xml:"name>first"`.
- CDATA content handling
- `,innerxml` flag handling in Marshal
- `,comment` flag handling
- `,any` flag handling

Feel free to add any of these features (or other features) using a pull request.

## Usage
```go
import "github.com/alexrsagen/go-libxml"
```

### Marshal
```go
func Marshal(v interface{}) (string, error)
```

This function returns the XML encoding of v.

See https://golang.org/pkg/encoding/xml/#Marshal for more information.

Example usage:
```go
package main

import (
	"encoding/xml"
	"fmt"

	"github.com/alexrsagen/go-libxml"
)

// Example source: https://github.com/golang/go/blob/master/src/encoding/xml/example_test.go
type address struct {
	City, State string
}
type person struct {
	XMLName   xml.Name `xml:"person"`
	ID        int      `xml:"id,attr"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	Age       int      `xml:"age"`
	Height    float32  `xml:"height,omitempty"`
	Married   bool
	address
}

func main() {
	v := &person{ID: 13, FirstName: "John", LastName: "Doe", Age: 42}
	v.address = address{"Hanga Roa", "Easter Island"}

	output, err := libxml.Marshal(v)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("output: %s\n", output)
}
```

### Unmarshal
```go
func Unmarshal(data string, v interface{}) error
```

This function parses the XML-encoded data and stores the result in the value pointed to by v, which must be an arbitrary struct, slice, or string. Well-formed data that does not fit into v is discarded.

See https://golang.org/pkg/encoding/xml/#Unmarshal for more information.

Example usage:
```go
package main

import (
	"encoding/xml"
	"fmt"

	"github.com/alexrsagen/go-libxml"
)

// Example source: https://github.com/golang/go/blob/master/src/encoding/xml/example_test.go
type address struct {
	City, State string
}
type person struct {
	XMLName   xml.Name `xml:"person"`
	ID        int      `xml:"id,attr"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	Age       int      `xml:"age"`
	Height    float32  `xml:"height,omitempty"`
	Married   bool
	address
}

func main() {
	v := &person{address: address{}}

	err := libxml.Unmarshal(`<?xml version="1.0" encoding="UTF-8"?>
<person id="13">
  <first_name>John</first_name>
  <last_name>Doe</last_name>
  <age>42</age>
  <Married>false</Married>
  <address>
    <City>Hanga Roa</City>
    <State>Easter Island</State>
  </address>
</person>`, v)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("v: %v\n", v)
}
```