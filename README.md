# Практическая работа №8 — Микросервисное приложение на Go

Этот проект — готовый пример выполнения практической работы №8 по дисциплине
«Разработка клиент-серверных приложений» (создание приложения с микросервисной архитектурой).fileciteturn0file0L120-L210

## Краткое описание предметной области

Приложение — упрощённый сервис заказов:

- **auth-service** — регистрация и вход пользователя, выдача JWT-токена (аналог OAuth2 Password flow).
- **user-service** — хранение профиля пользователя.
- **order-service** — создание и просмотр заказов.
- **gateway** — API Gateway, принимает все внешние запросы, проверяет JWT и проксирует запросы к нужным микросервисам.

Каждый бизнес‑микросервис использует **свою базу данных** (файл SQLite внутри контейнера).

## Соответствие требованиям задания

1. **Авторизация по OAuth2**  
   Используется JWT-токен, выдаваемый `auth-service` через эндпоинт `/auth/login`.  
   Gateway проверяет `Authorization: Bearer <token>` и пропускает запрос дальше только при валидном токене.

2. **Сервис конфигурации (аналог Spring Cloud Config)**  
   Конфигурация вынесена из кода в:
   - переменные окружения (задаются в манифестах Kubernetes),
   - YAML-манифесты Kubernetes, которые версионируются в Git.
   Это аналог централизованного конфигурационного сервиса.

3. **API Gateway**  
   Сервис `gateway` (Go) реализует паттерн API Gateway:
   - принимает все внешние запросы,
   - проверяет JWT,
   - маршрутизирует запросы по префиксам:
     - `/auth/...` → `auth-service`
     - `/users/...` → `user-service`
     - `/animes/...` → `anime-service`.

4. **Service Discovery**  
   Используется встроенный механизм Kubernetes:
   - каждый микросервис развёрнут как `Deployment` + `Service`,
   - обращение по DNS-имени сервиса (`http://auth-service:8080`, и т.д.).

5. **Балансировщик нагрузки**  
   Kubernetes `Service` (тип `ClusterIP` / `NodePort`) выступает балансировщиком между репликами Pod‑ов.

6. **Минимум 3 микросервиса предметной области**  
   Предметные микросервисы:
   - `auth-service`
   - `user-service`
   - `anime-service`

7. **Взаимодействие через FeignClient (аналог)**  
   В `anime-service` реализован типизированный HTTP‑клиент `UserClient` (пакет `internal/userclient`),  
   который ходит в `user-service`. Это Go‑аналог FeignClient.

8. **Работа с базой данных**  
   Все три сервиса (`auth-service`, `user-service`, `anime-service`) используют **свои отдельные SQLite‑БД**.

9. **Отдельный экземпляр БД для каждого микросервиса**  
   Для каждого сервиса задан отдельный путь к файлу БД через переменную окружения `DB_PATH`.

10. **Unit‑тесты**  
    В проекте есть простые unit‑тесты:
    - в `auth-service/internal/auth/jwt_test.go`
    - в `order-service/internal/userclient/client_test.go`

11. **Файлы для развёртывания (Kubernetes)**  
    В каталоге `k8s/` содержатся манифесты для всех сервисов и gateway.

12. **Развёртывание в Minikube (аналог Yandex/VK Cloud)**  
    В каталоге `scripts/` есть скрипты:
    - `deploy_minikube.sh` — сборка Docker‑образов и развёртывание в Minikube **одной командой**.
    - `delete_minikube.sh` — удаление ресурсов.

## Структура проекта

- `services/`
  - `gateway/` — API Gateway на Go
  - `auth-service/` — сервис авторизации
  - `user-service/` — сервис пользователей
  - `anime-service/` — сервис заказов
- `k8s/` — манифесты Kubernetes
- `scripts/` — скрипты развёртывания
- `README.md` — это описание

## Требования для запуска

- Установлен **Minikube**
- Установлен **kubectl**
- Установлен **Docker**
- Go не нужен на хосте — всё собирается внутри Docker.

## Быстрый старт (Minikube)

```bash
cd practicum8-go-microservices

# 1. Запустить minikube (если ещё не запущен)
minikube start

# 2. Развернуть всю систему в minikube одной командой
bash scripts/deploy_minikube.sh

# 3. Получить URL gateway
minikube service gateway -n microservices-pr8 --url
```

Пример взаимодействия:

1. **Регистрация пользователся**

```bash
curl -X POST "$GATEWAY_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com", "password":"secret"}'
```

2. **Логин (получение JWT)**

```bash
TOKEN=$(curl -s -X POST "$GATEWAY_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com", "password":"secret"}' | jq -r '.access_token')
```

3. **Получение профиля**

```bash
curl -H "Authorization: Bearer $TOKEN" "$GATEWAY_URL/users/me"
```

4. **добавления аниме**

```bash
curl -X POST "$GATEWAY_URL/orders" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"anime":"exampleName","folder_name":"watched"}'
```

5. **Просмотр своих заказов**

```bash
curl -H "Authorization: Bearer $TOKEN" "$GATEWAY_URL/animes/my"
```

Все команды выше выполняются **через gateway** и проходят полную цепочку микросервисов.
