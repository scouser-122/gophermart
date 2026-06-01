# cmd/gophermart

В данной директории будет содержаться код накопительной системы лояльности, который скомпилируется в бинарное
приложение.

Пример сборки и запуска из командной строки:

go build -o gophermart && ./gophermart \
-a localhost:45057  \
-l info \
-e dev \
-d="postgres://postgres:123@localhost:5432/gophermart_db?sslmode=disable"
