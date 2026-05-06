package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository/db"
	"github.com/scouser-122/gophermart/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Мок для pgx.Rows
type MockPostgresRowsOrder struct {
	mock.Mock
	orders []models.Order
	index  int
}

func (m *MockPostgresRowsOrder) Next() bool {
	return m.index < len(m.orders)
}

func (m *MockPostgresRowsOrder) Scan(dest ...interface{}) error {
	if m.index >= len(m.orders) {
		return errors.New("no more rows")
	}

	order := m.orders[m.index]
	*dest[0].(*string) = order.ID
	*dest[1].(*models.OrderStatus) = order.Status
	*dest[2].(*time.Time) = order.UploadedAt
	*dest[3].(**int64) = order.Accrual
	*dest[4].(*string) = order.UserLogin
	m.index++
	return nil
}

func (m *MockPostgresRowsOrder) Close()          {}
func (m *MockPostgresRowsOrder) Err() error      { return nil }
func (m *MockPostgresRowsOrder) Conn() *pgx.Conn { return &pgx.Conn{} }
func (m *MockPostgresRowsOrder) FieldDescriptions() []pgconn.FieldDescription {
	return []pgconn.FieldDescription{
		{Name: "id"},
		{Name: "status"},
		{Name: "uploaded_at"},
		{Name: "accrual"},
		{Name: "user_login"},
	}
}
func (m *MockPostgresRowsOrder) RawValues() [][]byte {
	return [][]byte{
		[]byte("id"),
		[]byte("status"),
		[]byte("uploaded_at"),
		[]byte("accrual"),
		[]byte("user_login"),
	}
}
func (m *MockPostgresRowsOrder) Values() ([]any, error) { return nil, fmt.Errorf("test err") }

func (m *MockPostgresRowsOrder) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 3")
}

func generateAuthToken(login string) string {
	serverConfig := config.DefaultServerConfig()
	jwtService := service.NewJwtService(&serverConfig)
	authToken, err := jwtService.GenerateJWT(login)
	if err != nil {
		panic(err)
	}
	return authToken
}

var orderUploadTests = []struct {
	name    string
	request request
	mockDB  db.MockPostgresDBTestData
	want    want
}{
	{
		name: "positive test upload order",
		request: request{
			method:      http.MethodPost,
			contentType: "text/plain",
			path:        "/api/user/orders",
			body:        `4242424242424242`,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockPool: &db.MockPostgresPool{
				MockMethods: func(tt db.MockPostgresDBTestData) {
					tt.MockPool.On("Query", mock.Anything, mock.Anything, mock.Anything).
						Return(
							&MockPostgresRowsOrder{
								orders: []models.Order{},
							},
							nil,
						)
					tt.MockPool.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(
						pgconn.NewCommandTag("INSERT 1"), nil,
					)
					tt.MockPool.On("Ping", mock.Anything).Return(nil)
				},
			},
		},
		want: want{
			code:        http.StatusAccepted,
			contentType: "application/json",
			body:        `{"status":"ok","message":"order successfully saved"}`,
		},
	},
	{
		name: "positive test upload order already uploaded",
		request: request{
			method:      http.MethodPost,
			contentType: "text/plain",
			path:        "/api/user/orders",
			body:        `4242424242424242`,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockPool: &db.MockPostgresPool{
				MockMethods: func(tt db.MockPostgresDBTestData) {
					tt.MockPool.On("Query", mock.Anything, mock.Anything, mock.Anything).
						Return(
							&MockPostgresRowsOrder{
								orders: []models.Order{
									{ID: "4242424242424242", Status: models.NewOrder, UploadedAt: time.Now(), Accrual: nil, UserLogin: "TestLogin"},
								},
							},
							nil,
						)
					tt.MockPool.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(
						pgconn.NewCommandTag("INSERT 1"), nil,
					)
					tt.MockPool.On("Ping", mock.Anything).Return(nil)
				},
			},
		},
		want: want{
			code:        http.StatusOK,
			contentType: "application/json",
			body:        `{"status":"ok","message":"order already uploaded"}`,
		},
	},
	{
		name: "negative test upload order bad request",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/orders",
			body:        ``,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockPool: &db.MockPostgresPool{
				MockMethods: func(tt db.MockPostgresDBTestData) {
					tt.MockPool.On("Ping", mock.Anything).Return(nil)
				},
			},
		},
		want: want{
			code:        http.StatusBadRequest,
			contentType: "application/json",
		},
	},
	{
		name: "negative test upload order unauthorized",
		request: request{
			method:      http.MethodPost,
			contentType: "text/plain",
			path:        "/api/user/orders",
			body:        `4242424242424242`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockPool: &db.MockPostgresPool{
				MockMethods: func(tt db.MockPostgresDBTestData) {
					tt.MockPool.On("Ping", mock.Anything).Return(nil)
				},
			},
		},
		want: want{
			code:        http.StatusUnauthorized,
			contentType: "application/json",
		},
	},
}

func TestOrderUpload(t *testing.T) {
	for _, test := range orderUploadTests {
		t.Run(test.name, func(t *testing.T) {
			r := createTestRouter(&test.mockDB)

			var bodyReader io.Reader
			if test.request.body != "" {
				jsonData := []byte(test.request.body)
				bodyReader = bytes.NewBuffer(jsonData)
			}

			request := httptest.NewRequest(test.request.method, test.request.path, bodyReader)
			request.Header.Add("Content-Type", test.request.contentType)
			if len(test.request.headers) > 0 {
				for k, v := range test.request.headers {
					request.Header.Add(k, v)
				}
			}

			// создаём новый Recorder
			w := httptest.NewRecorder()

			r.ServeHTTP(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
			if res.StatusCode == http.StatusOK && test.want.body != "" {
				bodyBytes, err := io.ReadAll(res.Body)
				assert.Nil(t, err)
				bodyString := string(bodyBytes)
				assert.Equal(t, test.want.body, strings.Replace(bodyString, "\n", "", -1))
			}
			if len(test.want.headersToBePresent) > 0 {
				for _, v := range test.want.headersToBePresent {
					resHeader := res.Header.Get(v)
					assert.NotEqual(t, "", resHeader, "Header %q is absent", v)
				}
			}
			res.Body.Close()
		})
	}
}
