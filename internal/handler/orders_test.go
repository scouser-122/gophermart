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
	"github.com/pashagolub/pgxmock/v5"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository/db"
	"github.com/scouser-122/gophermart/internal/service"
	"github.com/stretchr/testify/assert"
)

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
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow("4242424242424242", models.NewOrder, time.Now(), Ptr(float32(500)), nil, nil, "TestLogin"))
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
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow("4242424242424242", models.NewOrder, time.Now(), Ptr(float32(500)), nil, nil, "TestLogin"))
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
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow("4242424242424242", models.NewOrder, time.Now(), Ptr(float32(500)), nil, nil, "TestLogin2"))
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
	name            string
	request         request
	accrualResponse accrualResponse
	mockDB          db.MockPostgresDBTestData
	want            want
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
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"RPOCESSED","accrual":500}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow(
							"4242424242424242",
							models.NewOrder,
							time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
							nil, nil, nil,
							"TestLogin"))
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}))
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE orders SET").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectExec("UPDATE users SET balance").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectCommit()
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow(
							"4242424242424242",
							models.ProcessedOrder,
							time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
							Ptr(float32(500)),
							nil, nil,
							"TestLogin"))
			},
		},
		want: want{
			code:        http.StatusOK,
			contentType: "application/json",
			body:        `[{"number":"4242424242424242","status":"PROCESSED","uploaded_at":"2026-05-10T12:24:45+03:00","accrual":500}]`,
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
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}))
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

var userBalanceWithdrawTests = []struct {
	name            string
	request         request
	accrualResponse accrualResponse
	mockDB          db.MockPostgresDBTestData
	want            want
}{
	{
		name: "positive test user balance withdraw",
		request: request{
			method: http.MethodPost,
			path:   "/api/user/balance/withdraw",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
			body: `{"order":"4242424242424242","sum":200.5}`,
		},
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"RPOCESSING","accrual":500}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow(
							"4242424242424242",
							models.ProcessingOrder,
							time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
							nil, nil, nil,
							"TestLogin"))
				// mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
				// 	WithArgs("4242424242424242").
				// 	WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "user_login"}).
				// 		AddRow(
				// 			"4242424242424242",
				// 			models.ProcessingOrder,
				// 			time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
				// 			nil,
				// 			"TestLogin"))
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE orders SET").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectExec("UPDATE users SET balance").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectCommit()
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow(
							"4242424242424242",
							models.ProcessingOrder,
							time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
							Ptr(float32(500)),
							nil, nil,
							"TestLogin"))
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE users SET balance = balance - .+ WHERE login = .+2 AND balance >= .+").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectExec("UPDATE orders SET withdrawn").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectCommit()
			},
		},
		want: want{
			code:        http.StatusOK,
			contentType: "application/json",
		},
	},
	{
		name: "negative test user balance withdraw unauthorized",
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
		name: "negative test user balance withdraw balance not enough",
		request: request{
			method: http.MethodPost,
			path:   "/api/user/balance/withdraw",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
			body: `{"order":"4242424242424242","sum":200.5}`,
		},
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"RPOCESSING","accrual":500}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow(
							"4242424242424242",
							models.ProcessingOrder,
							time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
							nil, nil, nil,
							"TestLogin"))
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE orders SET").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectExec("UPDATE users SET balance").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectCommit()
				mock.ExpectQuery("SELECT .+ FROM orders WHERE id").
					WithArgs("4242424242424242").
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow(
							"4242424242424242",
							models.ProcessingOrder,
							time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
							Ptr(float32(500)),
							nil, nil,
							"TestLogin"))
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE users SET balance = balance - .+ WHERE login = .+2 AND balance >= .+").
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
		},
		want: want{
			code:        http.StatusPaymentRequired,
			contentType: "application/json",
		},
	},
	{
		name: "negative test user balance withdraw incorrect order number format",
		request: request{
			method: http.MethodPost,
			path:   "/api/user/balance/withdraw",
			body:   `{"order":"incorrect_oder_format_123","sum":200.5}`,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"RPOCESSING","accrual":500}`,
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
		name: "negative test user balance withdraw internal server error",
		request: request{
			method: http.MethodPost,
			path:   "/api/user/balance/withdraw",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
			body: `{"order":"4242424242424242","sum":200.5}`,
		},
		accrualResponse: accrualResponse{
			status: http.StatusOK,
			body:   `{"order":"4242424242424242","status":"RPOCESSING","accrual":500}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectBegin()
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
}

func TestUserBalanceWithdraw(t *testing.T) {
	for _, test := range userBalanceWithdrawTests {
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

var getUserWithdrawalsTest = []struct {
	name    string
	request request
	mockDB  db.MockPostgresDBTestData
	want    want
}{
	{
		name: "positive test get user withdrawals",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/withdrawals",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				// mock.ExpectQuery("SELECT id, withdrawn, processed_at FROM orders").
				// 	WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
				// 	WillReturnRows(mock.NewRows([]string{"id", "withdrawn", "processed_at"}).
				// 		AddRow(
				// 			"4242424242424242",
				// 			123.5,
				// 			time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60))))
				// mock.ExpectQuery("SELECT id, withdrawn, processed_at FROM orders").
				// 	WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
				// 	WillReturnError(pgx.ErrNoRows)
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}).
						AddRow(
							"4242424242424242",
							models.ProcessedOrder,
							time.Date(2026, 5, 10, 12, 24, 45, 0, time.FixedZone("", 3*60*60)),
							nil,
							Ptr(float32(123.5)),
							Ptr(time.Date(2026, 5, 10, 14, 24, 45, 0, time.FixedZone("", 3*60*60))),
							"TestLogin"))
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}))
			},
		},
		want: want{
			code:        http.StatusOK,
			contentType: "application/json",
			body:        `[{"order":"4242424242424242","sum":123.5,"processed_at":"2026-05-10T14:24:45+03:00"}]`,
		},
	},
	{
		name: "positive test get user withdrawals no data",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/withdrawals",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM orders WHERE user_login").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(mock.NewRows([]string{"id", "status", "uploaded_at", "accrual", "withdrawn", "processed_at", "user_login"}))
			},
		},
		want: want{
			code:        http.StatusNoContent,
			contentType: "application/json",
		},
	},
	{
		name: "negative test get user withdrawals unauthorized",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/withdrawals",
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
		name: "negative test get user withdrawals internal server error",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/withdrawals",
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

func TestGetUserWithdrawals(t *testing.T) {
	for _, test := range getUserWithdrawalsTest {
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
