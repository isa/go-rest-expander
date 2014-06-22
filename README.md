# Go REST Expander

A tiny library to allow your RESTful resources to be expanded and/or filtered. If you don't know what resource expansions are, I'd recommend reading [Linking and Resource Expansion](https://stormpath.com/blog/linking-and-resource-expansion-rest-api-tips/) from *StormPath* blog. Although I have a different implementation, it's still a good read...

My implementation is basically as following. Assuming you have below contact detail:

```
GET /contacts/id/3
```

call returns

```json
{
  "name": "John Doe",
  "cell": "+1 (123) 123-1233",
  "addresses": [
    "http://<your-domain>/addresses/id/147",
    "http://<your-domain>/addresses/id/412"
  ]
}
```

You can filter out the fields by calling:

```
GET /contacts/id/3?fields=name,cell
```

will return

```json
{
  "name": "John Doe",
  "cell": "+1 (123) 123-1233"
}
```

If you'd like to expand the addresses, just call:

```
GET /contacts/id/3?expand=addresses
```

will return

```json
{
  "name": "John Doe",
  "cell": "+1 (123) 123-1233",
  "addresses": [
    {
      "id": 147,
      "type": "home",
      "city": "Gotham City"
    },
    {
      "id": 412,
      "type": "business",
      "city": "Atlantis"
    }
  ]
}
```

If you want to filter out the expanded result as well, you could do this easily by:

```
GET /contacts/id/3?expand=addresses&filter=name,addresses(id,city)
```

will give you

```json
{
  "name": "John Doe",
  "addresses": [
    {
      "id": 147,
      "city": "Gotham City"
    },
    {
      "id": 412,
      "city": "Atlantis"
    }
  ]
}
```

As you can see, it's just my weekend project. So feel free to give feedback or open issues. I'll try my best to fix them in ASAP.

## Installation

```bash
go get github.com/isa/go-rest-expander/expander
```

## Usage

```go
package main

import (
   "github.com/isa/go-rest-expander/expander"
)

type Data struct {
  Name string
  Age int
}

func MyAwesomeHandler(request, response) {
   myData := Data{"John", 35}
   expansion := request.GetString("expand")
   fields := request.GetString("fields")

   myExpandedData := expander.Expand(myData, expansion, fields)

   // now you can serialize as json if you want
}

func main() {
   // assuming some REST service logic here
   // that uses above handler
}
```

## Developers

I use [GoConvey](http://goconvey.co/) for testing.

## License

Licensed under [Apache 2.0](LICENSE).
