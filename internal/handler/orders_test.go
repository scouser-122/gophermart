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
	"github.com/pashagolub/pgxmock/v5"
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
	name            string
	request         request
	accrualResponse accrualResponse
	mockDB          db.MockPostgresDBTestData
	want            want
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
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"NEW","accrual":500}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnError(pgx.ErrNoRows)
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO orders").
					WithArgs("4242424242424242", models.NewOrder, pgxmock.AnyArg(), pgxmock.AnyArg(), "TestLogin").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				mock.ExpectExec("UPDATE users SET balance").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectCommit()
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
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"NEW","accrual":500}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "user_login"}).
						AddRow("4242424242424242", models.NewOrder, time.Now(), Ptr(int64(500)), "TestLogin"))
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
			MockDBCalls: func(tt db.MockPostgresDBTestData) {},
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
			MockDBCalls: func(tt db.MockPostgresDBTestData) {},
		},
		want: want{
			code:        http.StatusUnauthorized,
			contentType: "application/json",
		},
	},

	{
		name: "negative test upload order already uploaded by another user",
		request: request{
			method:      http.MethodPost,
			contentType: "text/plain",
			path:        "/api/user/orders",
			body:        `4242424242424242`,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"NEW","accrual":500}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "user_login"}).
						AddRow("4242424242424242", models.NewOrder, time.Now(), Ptr(int64(500)), "TestLogin2"))
			},
		},
		want: want{
			code:        http.StatusConflict,
			contentType: "application/json",
			body:        `{"status":"ok","message":"order already uploaded by another user"}`,
		},
	},

	{
		name: "negative test upload order incorrect order number format",
		request: request{
			method:      http.MethodPost,
			contentType: "text/plain",
			path:        "/api/user/orders",
			body:        `incorrect_oder_format_123`,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {},
		},
		want: want{
			code:        http.StatusUnprocessableEntity,
			contentType: "application/json",
		},
	},

	{
		name: "negative test upload order internal server error db error",
		request: request{
			method:      http.MethodPost,
			contentType: "text/plain",
			path:        "/api/user/orders",
			body:        `4242424242424242`,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"NEW","accrual":500}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnError(fmt.Errorf("some error"))
			},
		},
		want: want{
			code:        http.StatusInternalServerError,
			contentType: "application/json",
		},
	},

	{
		name: "negative test upload order internal server error accrual error",
		request: request{
			method:      http.MethodPost,
			contentType: "text/plain",
			path:        "/api/user/orders",
			body:        `4242424242424242`,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		accrualResponse: accrualResponse{
			status: http.StatusInternalServerError,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {},
		},
		want: want{
			code:        http.StatusInternalServerError,
			contentType: "application/json",
		},
	},
}

func TestOrderUpload(t *testing.T) {
	for _, test := range orderUploadTests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				headers := rw.Header()
				headers.Add("Content-Type", "application/json")
				rw.WriteHeader(test.accrualResponse.status)
				if test.accrualResponse.body != "" {
					rw.Write([]byte(test.accrualResponse.body))
				}
			}))
			defer server.Close()

			r := createTestRouter(&test.mockDB, server.URL)

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

var getUserOrdersTests = []struct {
	name    string
	request request
	mockDB  db.MockPostgresDBTestData
	want    want
}{
	{
		name: "positive test get user orders",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/orders",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "user_login"}).
						AddRow(
							"4242424242424242",
							models.NewOrder,
							time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
							Ptr(int64(123)),
							"TestLogin"))
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
		},
		want: want{
			code:        http.StatusOK,
			contentType: "application/json",
			body:        `[{"id":"4242424242424242","status":"NEW","uploaded_at":"2026-05-10T12:24:45+03:00","accrual":"123"}]`,
		},
	},
	{
		name: "positive test get user orders no data",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/orders",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
		},
		want: want{
			code:        http.StatusNoContent,
			contentType: "application/json",
		},
	},
	{
		name: "negative test get user orders unauthorized",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/orders",
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {},
		},
		want: want{
			code:        http.StatusUnauthorized,
			contentType: "application/json",
		},
	},
	{
		name: "negative test get user orders internal server error",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/orders",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnError(fmt.Errorf("some error"))
			},
		},
		want: want{
			code:        http.StatusInternalServerError,
			contentType: "application/json",
		},
	},
}

func TestGetUserOrders(t *testing.T) {
	for _, test := range getUserOrdersTests {
		t.Run(test.name, func(t *testing.T) {
			r := createTestRouter(&test.mockDB, "")

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
