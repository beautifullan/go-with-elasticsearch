package main

import (
	"context"
	"github.com/olivere/elastic"
	"github.com/stretchr/testify/mock"
	"go-with-elasticsearch/create"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockClient struct {
	mock.Mock
}

func CheckBucketExists(name string, client *elastic.Client) bool {
	exists, err := client.IndexExists(name).Do(context.Background())
	if err != nil {
		panic(err)
	}
	return exists
}

func (m *MockClient) Index(index string, body string, docID string) (*http.Response, error) {
	// 使用 On 方法定义模拟的 Index 方法行为
	// 这里示例中假设当调用 Index 方法时，返回一个模拟的 http.Response 对象和 nil 错误
	args := m.Called(index, body, docID)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestIndexPaper(t *testing.T) {
	// 创建模拟的 Elasticsearch 客户端
	mockES := &MockClient{}

	// 设置预期的行为
	expectedResponse := &http.Response{} // 模拟的 http.Response 对象
	mockES.On("Index", "paper_index", "document_body", "document_id").Return(expectedResponse, nil)

	// 创建一个模拟的 HTTP 请求和响应
	req := httptest.NewRequest("POST", "http://localhost:9200/api/manage/index", nil)
	w := httptest.NewRecorder()

	// 调用创建索引的处理函数
	create.IndexHandler(w, req) // 传入模拟的 Elasticsearch 客户端

	// 检查响应状态码是否为 200
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// 可以根据具体情况进一步验证创建索引的功能是否按预期工作
}
