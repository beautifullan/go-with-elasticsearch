package search

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch"
	"github.com/elastic/go-elasticsearch/esapi"
	"go-with-elasticsearch/create"
	"log"
	"net/http"
	"net/url"
	"strings"
)

//var paper create.Paper

// SearchResponse 结构体表示搜索结果的格式
type SearchResponse struct {
	Total int            `json:"total"`
	Hits  []create.Paper `json:"hits"`
}

// searchPaper 函数执行Elasticsearch搜索操作
func searchPaper(es *elasticsearch.Client, query url.Values) (SearchResponse, error) {
	var searchResponse SearchResponse
	index := "paper_index"

	// 构建搜索查询
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{},
			},
		},
	}
	// 如果有查询参数，则根据查询参数构建查询
	if len(query) > 0 {
		for key, values := range query {
			if key == "title" || key == "abstract" || key == "content" || key == "tags" || key == "publishDate" {
				for _, value := range values {
					queryPart := map[string]interface{}{
						"match": map[string]interface{}{
							key: value,
						},
					}
					searchQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = append(
						searchQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]interface{}),
						queryPart,
					)
				}
			}
		}
	} else {
		// 如果没有查询参数，则构建一个匹配所有文档的查询
		searchQuery = map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
	}

	// 打印最终的 searchQuery

	//发送搜索请求，elasticsearch的api需要接收 json格式的数据作为请求体来执行搜索操作
	searchQueryJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return searchResponse, fmt.Errorf("error marshalling search query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  strings.NewReader(string(searchQueryJSON)),
	}
	///////
	//log.Printf("this is body %v ", strings.NewReader(string(searchQueryJSON)))
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return searchResponse, fmt.Errorf("error searching index: %w", err)
	}
	///////
	//log.Printf("this is res %v ", res)
	defer res.Body.Close()

	if res.IsError() {
		return searchResponse, fmt.Errorf("error searching index: %s", res.String())
	}

	//log.Println(res)
	// 解析搜索的结果
	//构建搜索查询时，要将查询对象转化为有效的 JSON 字符串而不是直接转换为字符串，不不然可能导致 Elasticsearch 在解析查询时出现问题。

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return searchResponse, fmt.Errorf("error decoding search response: %w", err)
	}

	// 提取命中结果和总记录数
	hits := response["hits"].(map[string]interface{})["hits"].([]interface{})
	searchResponse.Total = int(response["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	//for _, hit := range hits {
	//	hitMap := hit.(map[string]interface{})
	//	sourceMap := hitMap["_source"].(map[string]interface{})
	//	var paper create.Paper
	//	sourceJSON, err := json.Marshal(sourceMap)
	//	if err != nil {
	//		return searchResponse, fmt.Errorf("error marshalling source data: %w", err)
	//	}
	//	if err := json.Unmarshal(sourceJSON, &paper); err != nil {
	//		return searchResponse, fmt.Errorf("error unmarshalling paper data: %w", err)
	//	}
	//	//log.Printf("this is hit %v", hit)
	//	if paper.ID != 0 {
	//		searchResponse.Hits = append(searchResponse.Hits, paper)
	//	}
	//	log.Printf("this is id %v", paper.ID) //为什么打印出来对id为0呢
	//
	//	log.Printf("this is hit1111 % v", hits) //hits有但是searchresponse.hits为空
	//	log.Printf("this is hits %v", searchResponse.Hits)
	//}
	searchResponse.Total = 0
	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		sourceMap := hitMap["_source"].(map[string]interface{})
		var paper create.Paper

		sourceJSON, err := json.Marshal(sourceMap)
		if err != nil {
			return searchResponse, fmt.Errorf("error marshalling source data: %w", err)
		}
		if err := json.Unmarshal(sourceJSON, &paper); err != nil {
			return searchResponse, fmt.Errorf("error unmarshalling paper data: %w", err)
		}

		if paper.ID != 0 {
			searchResponse.Hits = append(searchResponse.Hits, paper)
			searchResponse.Total++
		} /////
		//log.Printf("this is hits %v", searchResponse.Hits)
	}

	return searchResponse, nil
}

// SearchHandler 函数处理搜索请求
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 创建Elasticsearch客户端
	cfg := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatal("elasticsearch connect failed")
		return
	}

	// 获取查询参数
	query := r.URL.Query()

	// 执行搜索
	searchResponse, err := searchPaper(es, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 编码并发送响应
	jsonResponse, err := json.Marshal(searchResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

//func main() {
//	r := mux.NewRouter()
//	r.HandleFunc("/api/search", SearchHandler).Methods(http.MethodGet)
//	log.Fatal(http.ListenAndServe(":8080", r))
//}
