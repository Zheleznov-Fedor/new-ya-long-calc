# Установка
0. Подготавливаемся  
   `go mod tidy`  
1. Для начала настроим настроим систему. 
В файле `.env` можно настроить следующие параметры:
    - TIME_"operation"  
	Время на выполнение каждой операции. 
	Операции: ADD - сложение, SUBSTRACT - вычитание, MULT - умножение, DIVISION - деление.
	- AGENTS_CNT  
	Количество вычислятовров
1. Начинаем запускаться.
   Находясь в директории проекта запустим следующие программы.
   Вычисляторы надо запускать в разных терминалах.
   Для каждого i-го вычислятора запуск будет выглядеть так (мы передаём ему порт, i начинается с 0):  
   `go run ./cmd/worker/main.go <5000 + i>`  
   Пример для 3 вычисляторов.
   ```
   ~ go run ./cmd/worker/main.go  5000
   ~ go run ./cmd/worker/main.go  5001
   ~ go run ./cmd/worker/main.go  5002
   ```  
   Дальше запустим орекстратор(он слушает на порту 8080)
   ```
   ~ go run ./cmd/server/main.go
   ```
Всё готово! Теперь перейдём к API

# API
Обрабатываются мат. выражения как с целыми числами, так и с числами с плавающей точкой. 
Доступные операции: +, -, *, /.
Скобок нет.  

Методы
- Регистрация   
  POST /register  
  Content-Type: application/json  
  {
    "login": ,
    "password":
  }
- Вход в аккаунт  
  POST /login  
  Content-Type: application/json  
  {
    "login": ,
    "password":
  }
  Вернёт JWT токен и попробует сохранить в куки(при использовании сопутствующего ПО).
  В curl-е будем передавать в виде header-а "auth-token"
- Добавить новое выражения    
  POST /expr  
  Content-Type: application/json  
  auth-token <JWT токен>    
  <Математическое выражение>
- Проверить готовность  
  GET /expr/<идентификатор выражения>  
  auth-token <JWT токен> 

# Примеры
- Регистрация:
  ```
  curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"login": "zhefed", "password": "ya-calc"}' \
  http://localhost:8080/register
  ```
- Вход в аккаунт:
  ```
  curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"login": "zhefed", "password": "ya-calc"}' \
  http://localhost:8080/login
  ```
  Вернётся токен, сохраним его для следующих шагов. В конце может стоять знак процента, его копировать не надо.
- Положить новое задание:
  ```
  curl -X POST \
  -H "Content-Type: application/json" \
  -H "auth-token: <токен полученный ранее>" \
  -d '"2 * 3 + 5 / 2"' \
  http://localhost:8080/expr
  ```
  ```
  curl -X POST \
  -H "Content-Type: application/json" \
  -H "auth-token: <токен полученный ранее>" \
  -d '"2*3 + 5.98/2 - 0.001 + 21.1"' \
  http://localhost:8080/expr
  ```
- Проверить готовность:
    ```
    curl -X GET http://localhost:8080/expr/123456789
    ```
