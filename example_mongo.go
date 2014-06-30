package main

import (
   "github.com/isa/go-rest-expander/expander"
   "fmt"
   "regexp"
   "encoding/json"
   "net/http"
   "labix.org/v2/mgo"
   "labix.org/v2/mgo/bson"
)

type Group struct {
   Name string
   Folks []mgo.DBRef
}

type Person struct {
   Id interface{} "_id"
   Name string
   Phone string
}

func groupHandler(w http.ResponseWriter, r *http.Request) {
   session, _ := mgo.Dial("localhost")
   defer session.Close()

   session.SetMode(mgo.Monotonic, true)
   g := session.DB("test").C("groups")

   group := Group{}
   _ = g.Find(bson.M{"name": "Family"}).One(&group)

   expander.ExpanderConfig = expander.Configuration{
      UsingMongo: true,
      IdURIs: map[string]string {
         "people": "http://localhost:9000/contacts/id",
      },
   }
   expanded := expander.Expand(group, r.FormValue("expand"), r.FormValue("filter"))
   result, _ := json.Marshal(expanded)
   fmt.Fprintf(w, string(result))
}

func contactHandler(w http.ResponseWriter, r *http.Request) {
   session, _ := mgo.Dial("localhost")
   defer session.Close()

   session.SetMode(mgo.Monotonic, true)
   c := session.DB("test").C("people")

   contact := Person{}
   _ = c.FindId(bson.ObjectIdHex(r.URL.Path[13:])).One(&contact)

   result, _ := json.Marshal(contact)
   fmt.Fprintf(w, string(result))
}

type route struct {
    pattern *regexp.Regexp
    handler http.Handler
}

type RegexpHandler struct {
    routes []*route
}

func (h *RegexpHandler) AddRoute(pattern string, handler func(http.ResponseWriter, *http.Request)) {
    h.routes = append(h.routes, &route{regexp.MustCompile(pattern), http.HandlerFunc(handler)})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    for _, route := range h.routes {
        if route.pattern.MatchString(r.URL.Path) {
            route.handler.ServeHTTP(w, r)
            return
        }
    }

    http.NotFound(w, r)
}

func main() {
   session, _ := mgo.Dial("localhost")
   defer session.Close()

   session.SetMode(mgo.Monotonic, true)
   c := session.DB("test").C("people")
   g := session.DB("test").C("groups")


   id1, id2 := bson.NewObjectId(), bson.NewObjectId()
   refs := []mgo.DBRef{mgo.DBRef{"people", id1, "test"}, mgo.DBRef{"people", id2, "test"}}
   _ = c.Insert(&Person{id1, "Ale", "+55 53 8116 9639"}, &Person{id2, "Cla", "+55 53 8402 8510"})
   _ = g.Insert(&Group{"Family", refs})

   go func() {
      regexMux := new(RegexpHandler)
      regexMux.AddRoute("/contacts/id/.*", contactHandler)
      regexMux.AddRoute("/groups/id/.*", groupHandler)
      http.ListenAndServe(":9000", regexMux)
   }()

   select {}
}
