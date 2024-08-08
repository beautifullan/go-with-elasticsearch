package create

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch"
	"github.com/elastic/go-elasticsearch/esapi"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"strings"
)

//不对呀不对 我还是需要先搞数据库 没有不行的还是,要根据数据库建索引的

// 定义字段，数据库中定义了数据的存储方式，但是没有定义在go中如何处理，这个结构体与表结构相匹配
type Paper struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Authors     string `json:"authors"`
	Abstract    string `json:"abstract"`
	Content     string `json:"content"`
	Tags        string `json:"tags"`
	Pdf         string `json:"pdf"`
	Publish     string `json:"publish"`
	PublishDate string `json:"publishDate"`
	Picture     string `json:"picture"`
}

type CreateIndexResponse struct {
	Message   string `json:"Message"`
	IndexName string `json:"IndexName"`
}

func indexPaper(db *sql.DB, es *elasticsearch.Client) CreateIndexResponse {
	// Create Index Request
	//mapping用于定义索引中的字段类型等属性
	index := "paper_index"

	//判断索引是否已经存在
	indexExistsReq := esapi.IndicesExistsRequest{
		Index: []string{index},
	}
	res, err := indexExistsReq.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("Error checking if index exists: %s", err)
	}
	defer res.Body.Close()
	log.Println(res.StatusCode)
	//if res.IsError() {
	//  log.Fatalf("Index checking error: %s", res.Status())
	//}

	if res.StatusCode == http.StatusOK {
		// 索引存在
		log.Println("index already exists")
		return CreateIndexResponse{
			Message:   fmt.Sprintf("Index %v already exists", index),
			IndexName: index,
		}
		//return CreateIndexResponse{
		//  Message:   fmt.Sprintf(" index %v exists", index),
		//  IndexName: index,
		//}
	} else if res.StatusCode == http.StatusNotFound {
		mapping := map[string]interface{}{
			"properties": map[string]interface{}{
				"title":       map[string]interface{}{"type": "text"},
				"authors":     map[string]interface{}{"type": "text"},
				"abstract":    map[string]interface{}{"type": "text"},
				"content":     map[string]interface{}{"type": "text"},
				"tags":        map[string]interface{}{"type": "text"},
				"pdf":         map[string]interface{}{"type": "text"},
				"publish":     map[string]interface{}{"type": "text"},
				"publishDate": map[string]interface{}{"type": "date"},
				"picture":     map[string]interface{}{"type": "text"},
			},
		}
		body := map[string]interface{}{
			"mappings": mapping, //mappings是定义字段映射关系的关键字，mappings:mapping创建索引时应该如何映射文档中的字段
		}
		// 将请求体转换为JSON
		requestBody, err := json.Marshal(body)
		if err != nil {
			log.Println(err)
		}
		// 创建索引请求,这里只是创建了
		req := esapi.IndexRequest{
			Index:      index,
			Body:       strings.NewReader(string(requestBody)),
			DocumentID: "", // 可以指定文档ID，这里为空表示由Elasticsearch生成
		}
		res, err := req.Do(context.Background(), es)
		if err != nil {
			log.Println(err)
		}
		defer res.Body.Close()

		if res.IsError() {
			log.Printf("error creating the index: %s", res.String())
		}
		rows, err := db.Query("SELECT * FROM paper")
		if err != nil {
			log.Println(err)
		}
		defer rows.Close()

		for rows.Next() {
			var paper Paper
			err := rows.Scan(&paper.ID, &paper.Title, &paper.Authors, &paper.Abstract, &paper.Content, &paper.Tags, &paper.Pdf, &paper.Publish, &paper.PublishDate, &paper.Picture)
			if err != nil {
				log.Println("Error scanning row:", err)
				continue
			}
			docID := paper.Title // 使用标题作为文档ID
			doc, err := json.Marshal(paper)
			if err != nil {
				log.Println("Error marshalling paper data:", err)
				continue
			}
			// 发送索引请求
			req := esapi.IndexRequest{
				Index:      index,
				Body:       strings.NewReader(string(doc)),
				DocumentID: docID,
			}
			res, err := req.Do(context.Background(), es)
			if err != nil {
				log.Fatalf("执行创建索引失败%v", err)
				//return CreateIndexResponse{
				//  Message:   fmt.Sprintf("create index %v failed", index),
				//  IndexName: index,
				//}

			}
			defer res.Body.Close()
		}

	}
	return CreateIndexResponse{
		Message:   fmt.Sprintf("create index %v successfully", index),
		IndexName: index,
	}

	//return CreateIndexResponse{}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	db, err := sql.Open("mysql", "paper:paper@tcp(localhost:3306)/paper")
	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()

	response := indexPaper(db, es)
	//if response.Message == fmt.Sprintf("create index %v failed", response.IndexName) {
	//  log.Println("创建索引失败")
	//  http.Error(w, "Failed to create index", http.StatusInternalServerError)
	//  return
	//}
	//if response.Message == fmt.Sprintf("create index %v successfully", response.IndexName) {
	//  log.Println("创建索引成功")
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
	//}

}

//func main() {
//  r := mux.NewRouter()
//  //http.HandleFunc("/api/manage/index", IndexHandler)
//  r.HandleFunc("/api/manage/index", IndexHandler).Methods(http.MethodPost)
//  log.Fatal(http.ListenAndServe(":8080", r))
//}
