package main

import (
   "expander"
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
