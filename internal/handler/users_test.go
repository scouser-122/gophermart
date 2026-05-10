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
	"github.com/scouser-122/gophermart/internal/repository/db"
	"github.com/stretchr/testify/assert"
)

var userRegisterTests = []struct {
	name    string
	request request
	mockDB  db.MockPostgresDBTestData
	want    want
}{
	{
		name: "positive test register user",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/register",
			body:        `{"login":"TestLogin","password":"TestPassword"}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM users WHERE login").
					WithArgs("TestLogin").
					WillReturnError(pgx.ErrNoRows)
				mock.ExpectExec("insert into users .+").
					WithArgs("TestLogin", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
		},
		want: want{
			code:               http.StatusOK,
			contentType:        "application/json",
			body:               `{"status":"ok","message":"user successfully registered"}`,
			headersToBePresent: []string{"Authorization"},
		},
	},
	{
		name: "negative test register user bad request",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/register",
			body:        `{"login":"TestLogin"}`,
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
		name: "negative test register user login already taken",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/register",
			body:        `{"login":"TestLogin","password":"TestPassword"}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM users WHERE login").
					WithArgs("TestLogin").
					WillReturnRows(mock.NewRows([]string{"login", "password", "balance", "created_at"}).
						AddRow("TestLogin", "TestPassword", 0.0, time.Now()))
			},
		},
		want: want{
			code:        http.StatusConflict,
			contentType: "application/json",
		},
	},
	{
		name: "negative test register user internal server error",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/register",
			body:        `{"login":"TestLogin","password":"TestPassword"}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM users WHERE login").
					WithArgs("TestLogin").
					WillReturnError(fmt.Errorf("some error"))
			},
		},
		want: want{
			code:        http.StatusInternalServerError,
			contentType: "application/json",
		},
	},
}

func TestUserRegister(t *testing.T) {
	for _, test := range userRegisterTests {
		t.Run(test.name, func(t *testing.T) {
			r := createTestRouter(&test.mockDB, "")

			var bodyReader io.Reader
			if test.request.body != "" {
				jsonData := []byte(test.request.body)
				bodyReader = bytes.NewBuffer(jsonData)
			}

			request := httptest.NewRequest(test.request.method, test.request.path, bodyReader)
			request.Header.Add("Content-Type", test.request.contentType)

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

var userLoginTests = []struct {
	name    string
	request request
	mockDB  db.MockPostgresDBTestData
	want    want
}{
	{
		name: "positive test login user",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/login",
			body:        `{"login":"TestLogin","password":"TestPassword"}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM users WHERE login").
					WithArgs("TestLogin").
					WillReturnRows(mock.NewRows([]string{"login", "password", "balance", "created_at"}).
						AddRow("TestLogin", "7bcf9d89298f1bfae16fa02ed6b61908fd2fa8de45dd8e2153a3c47300765328", 0.0, time.Now()))
			},
		},
		want: want{
			code:               http.StatusOK,
			contentType:        "application/json",
			body:               `{"status":"ok","message":"successfully logged in"}`,
			headersToBePresent: []string{"Authorization"},
		},
	},
	{
		name: "negative test login user bad request",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/login",
			body:        `{"login":"TestLogin"}`,
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
		name: "negative test login user failed",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/login",
			body:        `{"login":"TestLogin","password":"TestPassword"}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM users WHERE login").
					WithArgs("TestLogin").
					WillReturnRows(mock.NewRows([]string{"login", "password", "balance", "created_at"}).
						AddRow("TestLogin", "7bcf9d89298f1bfae16fa02ed6b6", 0.0, time.Now()))
			},
		},
		want: want{
			code:        http.StatusUnauthorized,
			contentType: "application/json",
		},
	},
	{
		name: "negative test register user internal server error",
		request: request{
			method:      http.MethodPost,
			contentType: "application/json",
			path:        "/api/user/register",
			body:        `{"login":"TestLogin","password":"TestPassword"}`,
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT .+ FROM users WHERE login").
					WithArgs("TestLogin").
					WillReturnError(fmt.Errorf("some error"))
			},
		},
		want: want{
			code:        http.StatusInternalServerError,
			contentType: "application/json",
		},
	},
}

func TestUserLogin(t *testing.T) {
	for _, test := range userLoginTests {
		t.Run(test.name, func(t *testing.T) {
			r := createTestRouter(&test.mockDB, "")

			var bodyReader io.Reader
			if test.request.body != "" {
				jsonData := []byte(test.request.body)
				bodyReader = bytes.NewBuffer(jsonData)
			}

			request := httptest.NewRequest(test.request.method, test.request.path, bodyReader)
			request.Header.Add("Content-Type", test.request.contentType)

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

var userGetBalanceTests = []struct {
	name    string
	request request
	mockDB  db.MockPostgresDBTestData
	want    want
}{
	{
		name: "positive test user get balance",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/balance",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT balance FROM users WHERE login").
					WithArgs("TestLogin").
					WillReturnRows(mock.NewRows([]string{"balance"}).AddRow(500.2))
				mock.ExpectQuery("SELECT SUM\\(withdrawn\\) FROM orders WHERE user_login").
					WithArgs("TestLogin").
					WillReturnRows(mock.NewRows([]string{"withdrawn"}).AddRow(Ptr(float32(60.25))))
			},
		},
		want: want{
			code:        http.StatusOK,
			contentType: "application/json",
			body:        `{"current":500.2,"withdrawn":60.25}`,
		},
	},
	{
		name: "negative test user get balance unauthorized",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/balance",
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
		name: "negative test user get internal server error",
		request: request{
			method: http.MethodGet,
			path:   "/api/user/balance",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", generateAuthToken("TestLogin")),
			},
		},
		mockDB: db.MockPostgresDBTestData{
			MockDBCalls: func(tt db.MockPostgresDBTestData) {
				mock := tt.PgxPoolIface
				mock.ExpectQuery("SELECT balance FROM users WHERE login").
					WithArgs("TestLogin").
					WillReturnError(fmt.Errorf("some error"))
			},
		},
		want: want{
			code:        http.StatusInternalServerError,
			contentType: "application/json",
		},
	},
}

func TestGetUserBalance(t *testing.T) {
	for _, test := range userGetBalanceTests {
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
