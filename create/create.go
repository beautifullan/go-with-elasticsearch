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
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	Abstract    string   `json:"abstract"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
	Pdf         string   `json:"pdf"`
	Publish     string   `json:"publish"`
	PublishDate string   `json:"publishDate"`
	Picture     string   `json:"picture"`
}

type CreateIndexResponse struct {
	Message   string `json:"Message"`
	IndexName string `json:"IndexName"`
}

var paper Paper

// 有变更返回true
//
//	func isDataChanged(db *sql.DB, es *elasticsearch.Client, index string) (bool, error) {
//		//从数据库读取和从es中读取 然后进行对比是否是一样的
//		rows, err := db.Query("SELECT id, title, authors, abstract, content, tags, pdf, publish, publishDate, picture FROM paper")
//		if err != nil {
//			return false, fmt.Errorf("select error %v", err)
//		}
//		defer rows.Close()
//
//		for rows.Next() {
//			var authorsJson, tagsJson sql.RawBytes
//			if err := rows.Scan(&paper.ID, &paper.Title, &authorsJson, &paper.Abstract, &paper.Content, &tagsJson, &paper.Pdf, &paper.Publish, &paper.PublishDate, &paper.Picture); err != nil {
//				return false, err
//			}
//			var authors []string
//			if err := json.Unmarshal(authorsJson, &authors); err != nil {
//				paper.Authors = authors
//				return false, fmt.Errorf("unmarshal authors error: %v", err)
//			}
//
//			var tags []string
//			if err := json.Unmarshal(tagsJson, &tags); err != nil {
//				paper.Tags = tags
//				return false, fmt.Errorf("unmarshal tags error: %v", err)
//			}
//			docID := paper.Title // 使用标题作为文档ID
//			req := esapi.GetRequest{
//				Index:      index,
//				DocumentID: docID,
//			}
//			res, err := req.Do(context.Background(), es)
//			if err != nil {
//				return false, fmt.Errorf("es request error: %v", err)
//			}
//			defer res.Body.Close()
//			if res.IsError() {
//				return false, fmt.Errorf("error getting document: %s", res.String())
//			}
//			var indexData map[string]interface{}
//			if err := json.NewDecoder(res.Body).Decode(&indexData); err != nil {
//				return false, fmt.Errorf("error decoding es response: %v", err)
//			}
//			// 比较数据库中的数据和索引中的数据是否相同
//			indexJSON, _ := json.Marshal(indexData["_source"])
//			dbJSON, _ := json.Marshal(paper)
//			if string(indexJSON) != string(dbJSON) {
//				log.Println("db data changed ")
//				return true, nil
//			}
//		}
//		log.Println("no data changed")
//		return false, nil
//
// }
func isDataChanged(db *sql.DB, es *elasticsearch.Client, index string) (bool, error) {
	rows, err := db.Query("SELECT id, title, authors, abstract, content, tags, pdf, publish, publishDate, picture FROM paper")
	if err != nil {
		return false, fmt.Errorf("select error %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var paper Paper
		var authorsJson, tagsJson sql.RawBytes
		if err := rows.Scan(&paper.ID, &paper.Title, &authorsJson, &paper.Abstract, &paper.Content, &tagsJson, &paper.Pdf, &paper.Publish, &paper.PublishDate, &paper.Picture); err != nil {
			return false, err
		}

		if err := json.Unmarshal(authorsJson, &paper.Authors); err != nil {
			return false, fmt.Errorf("unmarshal authors error: %v", err)
		}

		if err := json.Unmarshal(tagsJson, &paper.Tags); err != nil {
			return false, fmt.Errorf("unmarshal tags error: %v", err)
		}

		docID := paper.Title
		//docID := fmt.Sprintf("%v", paper.ID)
		req := esapi.GetRequest{
			Index:      index,
			DocumentID: docID,
		}
		res, err := req.Do(context.Background(), es)
		if err != nil {
			return false, fmt.Errorf("es request error: %v", err)
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusNotFound {
			// Document not found, treat as data changed
			log.Printf("Document ID %s not found in index", docID)
			return true, nil
		}

		if res.IsError() {
			return false, fmt.Errorf("error getting document: %s", res.String())
		}

		var indexData map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&indexData); err != nil {
			return false, fmt.Errorf("error decoding es response: %v", err)
		}

		//indexJSON, _ := json.Marshal(indexData["_source"])
		//dbJSON, _ := json.Marshal(paper)
		//if string(indexJSON) != string(dbJSON) {
		//	log.Println("db data changed")
		if !comparePaperWithIndexData(paper, indexData["_source"].(map[string]interface{})) {
			log.Printf("db data changed for document ID %s", docID)
			return true, nil
		}

	}
	log.Println("no data changed")
	return false, nil
}
func comparePaperWithIndexData(paper Paper, indexData map[string]interface{}) bool {
	// 逐字段比较
	if paper.Title != indexData["title"] ||
		!compareStringSlices(paper.Authors, indexData["authors"].([]interface{})) ||
		paper.Abstract != indexData["abstract"] ||
		paper.Content != indexData["content"] ||
		!compareStringSlices(paper.Tags, indexData["tags"].([]interface{})) ||
		paper.Pdf != indexData["pdf"] ||
		paper.Publish != indexData["publish"] ||
		paper.PublishDate != indexData["publishDate"] ||
		paper.Picture != indexData["picture"] {
		return false
	}
	return true
}

func compareStringSlices(a []string, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i].(string) {
			return false
		}
	}
	return true
}

// 判断索引内是否有内容,有数据则是true
func isIndexFilled(es *elasticsearch.Client, index string) (bool, error) {
	countReq := esapi.CountRequest{
		Index: []string{index},
	}
	countRes, err := countReq.Do(context.Background(), es)
	if err != nil {
		return false, fmt.Errorf("error counting documents in index: %s", err)
	}
	defer countRes.Body.Close()

	if countRes.IsError() {
		return false, fmt.Errorf("error counting documents: %s", countRes.String())
	}

	var countResult map[string]interface{}
	if err := json.NewDecoder(countRes.Body).Decode(&countResult); err != nil {
		return false, fmt.Errorf("error parsing count response: %s", err)
	}

	if countResult["count"].(float64) > 0 {
		return true, nil
	}
	return false, nil
}
func importDataFromDB(db *sql.DB, es *elasticsearch.Client, index string) {
	// 查询数据库以获取数据
	rows, err := db.Query("SELECT id, title, authors, abstract, content, tags, pdf, publish, publishDate, picture FROM paper")
	if err != nil {
		log.Printf("select error %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var paper Paper
		var authorsJson, tagsJson sql.RawBytes

		// 扫描行并处理错误
		if err := rows.Scan(&paper.ID, &paper.Title, &authorsJson, &paper.Abstract, &paper.Content, &tagsJson, &paper.Pdf, &paper.Publish, &paper.PublishDate, &paper.Picture); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// 解析 JSON 数据
		if err := json.Unmarshal(authorsJson, &paper.Authors); err != nil {
			log.Printf("Error unmarshalling authors: %v", err)
			continue
		}

		if err := json.Unmarshal(tagsJson, &paper.Tags); err != nil {
			log.Printf("Error unmarshalling tags: %v", err)
			continue
		}

		// 将 Paper 结构体转换为 JSON
		doc, err := json.Marshal(paper)
		if err != nil {
			log.Printf("Error marshalling paper data: %v", err)
			continue
		}

		// 生成文档 ID（可以使用 paper.ID 或其他唯一字段）
		//docID := fmt.Sprintf("%v", paper.Title)
		docID := fmt.Sprintf("%v", paper.ID)

		// 创建索引请求
		req := esapi.IndexRequest{
			Index:      index,
			Body:       strings.NewReader(string(doc)),
			DocumentID: docID,
		}

		// 执行请求并处理错误
		res, err := req.Do(context.Background(), es)
		if err != nil {
			log.Printf("Error indexing document ID %v: %v", docID, err)
			continue
		}
		defer res.Body.Close()

		if res.IsError() {
			log.Printf("Error response for document ID %s: %s", docID, res.String())
		}
	}
}

func updateDataInIndex(db *sql.DB, es *elasticsearch.Client, index string) {
	// 查询数据库以获取数据
	rows, err := db.Query("SELECT id, title, authors, abstract, content, tags, pdf, publish, publishDate, picture FROM paper")
	if err != nil {
		log.Fatalf("error querying database")
	}
	defer rows.Close()

	for rows.Next() {
		var paper Paper
		var authorsJson, tagsJson sql.RawBytes

		// 扫描行并处理错误
		if err := rows.Scan(&paper.ID, &paper.Title, &authorsJson, &paper.Abstract, &paper.Content, &tagsJson, &paper.Pdf, &paper.Publish, &paper.PublishDate, &paper.Picture); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// 解析 JSON 数据
		var authors []string
		if err := json.Unmarshal(authorsJson, &authors); err != nil {
			log.Fatal(err)
		}
		paper.Authors = authors

		var tags []string
		if err := json.Unmarshal(tagsJson, &tags); err != nil {
			log.Fatal(err)
		}
		paper.Tags = tags

		// 将 Paper 结构体转换为 JSON
		doc, err := json.Marshal(paper)
		if err != nil {
			log.Printf("Error marshalling paper  %v", err)
			continue
		}

		// 生成文档 ID（可以使用 paper.ID 或其他唯一字段）
		docID := fmt.Sprintf("%v", paper.Title)

		// 创建索引请求
		req := esapi.IndexRequest{
			Index:      index,
			Body:       strings.NewReader(string(doc)),
			DocumentID: docID,
		}

		// 执行请求并处理错误
		res, err := req.Do(context.Background(), es)
		if err != nil {
			log.Printf("Error indexing document ID %s: %v", docID, err)
			continue
		}
		defer res.Body.Close()

		if res.IsError() {
			log.Printf("Error response for document ID %s: %s", docID, res.String())
		} else {
			log.Printf("Paper ID %s updated successfully", paper.Title)
		}
	}
}

func indexPaper(db *sql.DB, es *elasticsearch.Client) CreateIndexResponse {
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
	if res.StatusCode == http.StatusOK {
		// 索引存在
		log.Println("index already exists")
		//存在我就要看index中有没有数据
		isFilled, err := isIndexFilled(es, index)
		if err != nil {
			log.Fatalf("error checking if index is filled %s", err)
		}
		if !isFilled {
			log.Println("index exists but has no documents, importing data...")
			importDataFromDB(db, es, index)
			return CreateIndexResponse{
				Message:   fmt.Sprintf("index %v exists but was empty, data imported", index),
				IndexName: index,
			}
		}
		changed, err := isDataChanged(db, es, index)
		if err != nil {
			log.Fatalf("Error checking if data changed: %s", err)
		}
		if changed {
			log.Println("data has changed, updating index...")
			updateDataInIndex(db, es, index)
			return CreateIndexResponse{
				Message:   fmt.Sprintf("Update index %v successfully", index),
				IndexName: index,
			}
		}
		return CreateIndexResponse{
			Message:   fmt.Sprintf("Index %v already exists and index content not changed", index),
			IndexName: index,
		}

	} else if res.StatusCode == http.StatusNotFound {
		mapping := map[string]interface{}{
			"properties": map[string]interface{}{
				"title":       map[string]interface{}{"type": "text", "index": "true"},
				"authors":     map[string]interface{}{"type": "text"},
				"abstract":    map[string]interface{}{"type": "text", "index": "true"},
				"content":     map[string]interface{}{"type": "text", "index": "true"},
				"tags":        map[string]interface{}{"type": "text", "index": "true"},
				"pdf":         map[string]interface{}{"type": "text"},
				"publish":     map[string]interface{}{"type": "text"},
				"publishDate": map[string]interface{}{"type": "date", "index": "true"},
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
		//req := esapi.IndexRequest{
		//	Index:      index,
		//	Body:       strings.NewReader(string(requestBody)),
		//	DocumentID: "", // 可以指定文档ID，这里为空表示由Elasticsearch生成
		//}
		//res, err := req.Do(context.Background(), es)
		//if err != nil {
		//	log.Println(err)
		//}
		//defer res.Body.Close()
		//
		//if res.IsError() {
		//	log.Printf("error creating the index: %s", res.String())
		//}
		////rows, err := db.Query("SELECT title,authors, abstract, content,tags, pdf, publish, publishDate, picture FROM paper")
		//rows, err := db.Query("SELECT id,title,JSON_EXTRACT(authors,'$') AS authors, abstract, content,JSON_EXTRACT(tags,'$') AS tags, pdf, publish, publishDate, picture FROM paper")
		//if err != nil {
		//	log.Println(err)
		//}
		//defer rows.Close()
		//
		//for rows.Next() {
		//	var paper Paper
		//	var authorsJson, tagsJson sql.RawBytes
		//	//err := rows.Scan(&paper.Title, &paper.Authors, &paper.Abstract, &paper.Content, &paper.Tags, &paper.Pdf, &paper.Publish, &paper.PublishDate, &paper.Picture)
		//	//if err != nil {
		//	//	log.Println("Error scanning row:", err)
		//	//	continue
		//	//}
		//	if err := rows.Scan(&paper.ID, &paper.Title, &authorsJson, &paper.Abstract, &paper.Content, &tagsJson, &paper.Pdf, &paper.Publish, &paper.PublishDate, &paper.Picture); err != nil {
		//		log.Fatal(err)
		//	}
		//	var authors []string
		//	if err := json.Unmarshal(authorsJson, &authors); err != nil {
		//		log.Fatal(err)
		//	}
		//	paper.Authors = authors
		//
		//	var tags []string
		//	if err := json.Unmarshal(tagsJson, &tags); err != nil {
		//		log.Fatal(err)
		//	}
		//	paper.Tags = tags
		//	if err := json.Unmarshal(authorsJson, &paper.Authors); err != nil {
		//		log.Fatal(err)
		//	}
		//	if err := json.Unmarshal(tagsJson, &paper.Tags); err != nil {
		//		log.Fatal(err)
		//	}
		//	//fmt.Println("Authors:", paper.Authors)
		//	//fmt.Println("Tags:", paper.Tags)
		//	//t := reflect.TypeOf(paper.Authors)
		//	//log.Println(t)
		//
		//	docID := paper.Title // 使用标题作为文档ID
		//	doc, err := json.Marshal(paper)
		//	if err != nil {
		//		log.Println("Error marshalling paper data:", err)
		//		continue
		//	}
		//	//log.Println(paper)
		//	//log.Println(paper.Authors) ["Jane Doe", "John Smith"]
		//	// 发送索引请求
		//	req := esapi.IndexRequest{
		//		Index:      index,
		//		Body:       strings.NewReader(string(doc)),
		//		DocumentID: docID,
		//	}
		//	log.Println(strings.NewReader(string(doc)))
		//	res, err := req.Do(context.Background(), es)
		//	if err != nil {
		//		log.Fatalf("执行创建索引失败%v", err)
		//		//return CreateIndexResponse{
		//		//  Message:   fmt.Sprintf("create index %v failed", index),
		//		//  IndexName: index,
		//		//}
		//
		//	}
		//	defer res.Body.Close()

		CreateIndexReq := esapi.IndexRequest{
			Index:      index,
			Body:       strings.NewReader(string(requestBody)),
			DocumentID: "",
		}
		createRes, err := CreateIndexReq.Do(context.Background(), es)
		if err != nil {
			log.Fatalf("Error creating index: %s", err)
		}
		defer createRes.Body.Close()
		if createRes.IsError() {
			log.Fatalf("error creating index:%s", createRes.String())
		}
		log.Println("Index created successfully, importing data...")
		importDataFromDB(db, es, index)
		return CreateIndexResponse{
			Message:   fmt.Sprintf("create index %v and imported data", index),
			IndexName: index,
		}

	}
	return CreateIndexResponse{
		Message:   fmt.Sprintf("Index %v processed successfully", index),
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
