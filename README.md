# Go REST Expander

A tiny library to allow your RESTful resources to be expanded and/or filtered. If you don't know what resource expansions are, I'd recommend reading [Linking and Resource Expansion](https://stormpath.com/blog/linking-and-resource-expansion-rest-api-tips/) from *StormPath* blog. Although I have a different implementation, it's still a good read...

[![wercker status](https://app.wercker.com/status/9daffd357de5205f4f9e5f185dda36a8/s "wercker status")](https://app.wercker.com/project/bykey/9daffd357de5205f4f9e5f185dda36a8)

My implementation is basically as following. Assuming you have below contact detail:

```
GET http://localhost:9003/contacts/id/3
```

call returns

```json
{
  "addresses": [
    {
      "ref": "http://localhost:9002/addresses/id/147",
      "rel": "home",
      "verb": "GET"
    },
    {
      "ref": "http://localhost:9002/addresses/id/412",
      "rel": "business",
      "verb": "GET"
    }
  ],
  "cell": "+1 (312) 888-44444",
  "group": {
    "ref": "http://localhost:9001/groups/id/7",
    "rel": "family",
    "verb": "GET"
  },
  "id": 3,
  "name": "John Doe"
}
```

You can filter out the fields by calling:

```
GET http://localhost:9003/contacts/id/3?filter=name,cell
```

will return

```json
{
  "cell": "+1 (312) 888-44444",
  "name": "John Doe"
}
```

If you'd like to expand the addresses, just call:

```
GET http://localhost:9003/contacts/id/3?expand=addresses
```

will return

```json
{
  "addresses": [
    {
      "city": {
        "ref": "http://localhost:9003/cities/id/1",
        "rel": "home",
        "verb": "GET"
      },
      "id": 147
    },
    {
      "city": {
        "ref": "http://localhost:9003/cities/id/2",
        "rel": "business",
        "verb": "GET"
      },
      "id": 412
    }
  ],
  "cell": "+1 (312) 888-44444",
  "group": {
    "ref": "http://localhost:9001/groups/id/7",
    "rel": "family",
    "verb": "GET"
  },
  "id": 3,
  "name": "John Doe"
}
```

If you want to filter out the expanded result as well, you could do this easily by:

```
GET http://localhost:9003/contacts/id/3?expand=addresses&filter=name,addresses(id,city)
```

will give you

```json
{
  "addresses": [
    {
      "city": {
        "ref": "http://localhost:9003/cities/id/1",
        "rel": "home",
        "verb": "GET"
      },
      "id": 147
    },
    {
      "city": {
        "ref": "http://localhost:9003/cities/id/2",
        "rel": "business",
        "verb": "GET"
      },
      "id": 412
    }
  ],
  "name": "John Doe"
}
```

If you wanna go nuts, you can always try something like:

```
GET http://localhost:9003/contacts/id/3?expand=*&filter=name,addresses(city(name))
```

You'll get:

```json
{
  "addresses": [
    {
      "city": {
        "name": "Gotham City"
      }
    },
    {
      "city": {
        "name": "Atlantis"
      }
    }
  ],
  "name": "John Doe"
}
```

Filter default is showing all results, and expansion default is expanding nothing. If you wanna expand everything try `*` for it.

As you can see, it's just my weekend project. So feel free to give feedback or open issues. I'll try my best to fix them in ASAP.

## Mongo DBRef Expansions

I also added a functionality for expanding mongo DBRef fields as well. So if you are using `mgo`, you can easily expand and resolve the Mongo references as well. To do so, you need to set the configuration like:

```go
expander.ExpanderConfig = expander.Configuration{
   UsingMongo: true,
   IdURIs: map[string]string {
      "people": "http://localhost:9000/contacts/id",
   },
}
```

Here `IdURIs` is basically a map of collection -> base URI of your resources. I added another example named `example_mongo` in the project directory. You can check it as well for understanding how it works. It's pretty simple.

By default, it won't expand mongo references.

## Installation

```bash
go get github.com/isa/go-rest-expander/expander
```

## Usage

Basically all you need to do is calling the Expand function with the right filters and expansion parameter.

```go
expanded := expander.Expand(myData, expansion, filter)
```

That's it. You can always check the `example.go` file in the root directory for a running example. Just run the example by:

```bash
go run example.go
```

This will create 4 dummy endpoints (contacts, addresses, cities, groups). Basically the above examples. You can use your favorite browser or curl to try above examples on the same endpoints.

Or if you are lazy like me, just copy paste following and then you are good to go.

```go
package main

import (
   "github.com/isa/go-rest-expander/expander"
   "fmt"
   "encoding/json"
   "net/http"
)

type Contact struct {
   Id int `json:"id"`
   Name string `json:"name"`
   Cell string `json:"cell"`
   Group Link `json:"group"`
   Addresses []Link `json:"addresses"`
}

type Address struct {
   Id int `json:"id"`
   City Link `json:"city"`
}

type City struct {
   Id int `json:"id"`
   Name string `json:"name"`
}

type Group struct {
   Id int `json:"id"`
   Name string `json:"name"`
   Description string `json:"description"`
}

type Link struct {
   Ref string `json:"ref"`
   Rel string `json:"rel"`
   Verb string `json:"verb"`
}

func contactHandler(w http.ResponseWriter, r *http.Request) {
   g := Link{"http://localhost:9001/groups/id/7", "family", "GET"}
   a1 := Link{"http://localhost:9002/addresses/id/147", "home", "GET"}
   a2 := Link{"http://localhost:9002/addresses/id/412", "business", "GET"}

   c := Contact{3, "John Doe", "+1 (312) 888-44444", g, []Link{a1, a2}}

   expansion, filter := r.FormValue("expand"), r.FormValue("filter")
   expanded := expander.Expand(c, expansion, filter)
   result, _ := json.Marshal(expanded)

   fmt.Fprintf(w, string(result))
}

func groupHandler(w http.ResponseWriter, r *http.Request) {
   g := Group{7, "Family", "My family members"}
   result, _ := json.Marshal(g)

   fmt.Fprintf(w, string(result))
}

func addressHandler(w http.ResponseWriter, r *http.Request) {
   addresses := map[string]Address {
      "147": Address{147, Link{"http://localhost:9003/cities/id/1", "home", "GET"}},
      "412": Address{412, Link{"http://localhost:9003/cities/id/2", "business", "GET"}},
   }
   result, _ := json.Marshal(addresses[r.URL.Path[14:]])

   fmt.Fprintf(w, string(result))
}

func cityHandler(w http.ResponseWriter, r *http.Request) {
   cities := map[string]City {
      "1": City{1, "Gotham City"},
      "2": City{2, "Atlantis"},
   }
   result, _ := json.Marshal(cities[r.URL.Path[11:]])

   fmt.Fprintf(w, string(result))
}

func main() {
   go func() {
      http.HandleFunc("/contacts/id/3", contactHandler)
      http.ListenAndServe(":9000", nil)
   }()

   go func() {
      http.HandleFunc("/groups/id/7", groupHandler)
      http.ListenAndServe(":9001", nil)
   }()

   go func() {
      http.HandleFunc("/addresses/id/147", addressHandler)
      http.HandleFunc("/addresses/id/412", addressHandler)
      http.ListenAndServe(":9002", nil)
   }()

   go func() {
      http.HandleFunc("/cities/id/1", cityHandler)
      http.HandleFunc("/cities/id/2", cityHandler)
      http.ListenAndServe(":9003", nil)
   }()

   select {}
}

```
## Caching

Expander includes a caching mechanism to cache HTTP-calls. It is **deactivated by default**, to activate it just add the following to your configuration:

```go
expander.ExpanderConfig = expander.Configuration{
   UsingCache: true,
   CacheExpInSeconds: 86400, // 24h
   ConnectionTimeoutInS = 2,
   ...
}
```
CacheExpInSeconds is the maximum time that an entry is cached.

## Developers

I use [GoConvey](http://goconvey.co/) for testing.

## License

Licensed under [Apache 2.0](LICENSE).
