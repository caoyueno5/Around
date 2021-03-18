package main

import (
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
	elastic "gopkg.in/olivere/elastic.v3"
	"log"
	"net/http"
	"reflect"
	"strconv"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Post struct {
	User     string `json:"user"`
	Message  string `son:"message"`
	Location string `json:"location"`
}

const (
	DISTANCE = "200km"
	INDEX    = "around"
	TYPE     = "post"
	ES_URL   = "http://http://34.68.138.55:9200"
)

func main() {
	clinet, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}

	exists, err := client.IndexExists(INDEX).Do()
	if err != nil {
		panic(err)
	}
	if !exists {
		mapping := `{
			"mappings":{
				"post":{
					"properties":{
						"location":{
							"type":"geo_point"
						}
					}
				}
			}
		}`
	}
	_, err = client.CreateIndex(INDEX).Body(mapping).Do()
	if err != nil {
		panic(err)
	}
	fmt.Println("started-service")
	http.HandleFunc("/post", handlerPost)
	log.Fatal(http.ListenAndServe(":8080", nil))
	http.HandleFunc("/search", handlerSearch)
}

func handlerPost(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one post request")

	decoder := json.NewDecoder(r.Body)
	var p Post
	if err := decoder.Decode(&p); err != nil {
		panic(err)
		return
	}
	fmt.Fprintf(w, "Post received: %s\n", p.Message)

	id := uuid.New()
	saveToES(&p, id)
}

func saveToES(p *Post, id string) {
	es_client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
	}
	_, err = es_client.Index().
		Index(INDEX).
		Type(TYPE).
		Id(id).
		BodyJson(p).
		Refresh(true).
		Do()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Post is saved to index: %s\n", p.Message)
}

func handlerSearch(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Reveiced one request for search.")
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	ran := DISTANCE
	if val := r.URL.Query().Get("range"); val != "" {
		ran = val + "km"
	}
	fmt.Println("range is", ran)

	fmt.Printf("Search received: %f %f %s\n", lat, lon, ran)
	client, err := elastic.NewClient(elastic.SetURL(ES-URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
	}
	q := elastic.NewGeoDistanceQuery("location")
	q = q.Distance(ran).Lat(lat).Lon(lon)
	searchResult, err := client.Search().
		Index(INDEX).
		Query(q).
		Pretty(true).
		Do()
	if err != nil {
		panic(err)
	}
	fmt.Println("Query took %d millisenconds\n", searchResult.TookInMillis)
	fmt.Printf("Found a total of %d posts\n", searchResult.TotalHits())

	var typ Post
	var ps Post
	for _, item := range searchResult.Each(reflect.TypeOf(typ)) { // instance of
		p := item.(Post) // p = (Post) item
		fmt.Printf("Post by %s: %s at lat %v and lon %v\n", p.User, p.Message, p.Location.Lat, p.Location.Lon)

		//TODO: filter restricted words
		ps = append(ps, p)
	}

	js, err := json.Marshal(ps)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(js)
}
