package main

import (
	"github.com/gorilla/mux"
	"go-with-elasticsearch/create"
	"go-with-elasticsearch/search"
	"log"
	"net/http"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/manage/index", create.IndexHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/search", search.SearchHandler).Methods(http.MethodGet)
	log.Fatal(http.ListenAndServe(":8080", r))

}

//package main
//
//import (
//	"context"
//	"github.com/elastic/go-elasticsearch"
//	"github.com/elastic/go-elasticsearch/esapi"
//	"log"
//)
//
//func main() {
//	cfg := elasticsearch.Config{
//		Addresses: []string{"http://localhost:9200"},
//	}
//	es, err := elasticsearch.NewClient(cfg)
//	if err != nil {
//		log.Fatalf("Error creating the client: %s", err)
//	}
//
//	req := esapi.IndicesDeleteRequest{
//		Index: []string{"paper_index"},
//	}
//	res, err := req.Do(context.Background(), es)
//	if err != nil {
//		log.Fatalf("Error getting response: %s", err)
//	}
//	defer res.Body.Close()
//	if res.IsError() {
//		log.Fatalf("Error: %s", res.String())
//	}
//
//	log.Println("Index deleted successfully")
//}
