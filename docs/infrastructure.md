# Redis, Kubernetes и мониторинг

Эта инфраструктура позволяет запустить несколько экземпляров мессенджера и наблюдать их работу. Для локального запуска достаточно Docker Compose. Kubernetes-манифесты рассчитаны на учебный кластер с default StorageClass, например kind или minikube.

## Зачем здесь Redis

- **Pub/Sub:** каждый экземпляр Go держит только свои WebSocket-соединения. Событие доставляется локальным клиентам и публикуется в Redis; остальные экземпляры доставляют его своим адресатам. Идентификатор отправителя предотвращает повторную доставку в исходном экземпляре. Работают сообщения, изменения, реакции, read status и личный mute.
- **Общий rate limit:** атомарный Lua-скрипт и sorted set разрешают одному пользователю 10 отправок за скользящие 5 секунд суммарно через REST/WebSocket на всех репликах. Время берётся у Redis, ключи автоматически истекают.
- **PostgreSQL остаётся источником данных:** Redis не хранит историю чатов. Pub/Sub имеет [доставку at-most-once](https://redis.io/docs/latest/develop/pubsub/), поэтому после разрыва соединения клиент восстанавливает историю, реакции и mute через REST.

Все экземпляры одного окружения должны использовать одинаковые PostgreSQL, JWT_SECRET, REDIS_URL и REDIS_PREFIX. Разным окружениям задавай разные префиксы: номер Redis DB **не изолирует Pub/Sub-каналы**.

Если REDIS_URL пустой, остаются локальные Hub и rate limiter: такой режим подходит для одного экземпляра. Compose и Kubernetes включают Redis автоматически. При недоступности настроенного Redis readiness возвращает 503, отправка новых сообщений — 503 (лимит нельзя обойти). Уже сохранённые данные не откатываются, локальная доставка при ошибке публикации продолжается; ошибки публикации считаются в метрике.

Redis здесь без persistence: перезапуск обнуляет временные лимиты, события во время недоступности не воспроизводятся. Клиент Redis переподключается автоматически. Это не transactional outbox и не гарантия exactly-once.

## Быстрый запуск с Grafana

Из корня репозитория, после создания .env из .env.example:

```bash
docker compose --profile monitoring up -d --build
curl http://localhost:8080/ready
```

- API: http://localhost:8080
- Prometheus: http://localhost:9091 — разделы Targets и Alerts.
- Grafana: http://localhost:3000 — пользователь admin, пароль из GRAFANA_ADMIN_PASSWORD (демо-значение messenger-demo).
- Дашборд **Messenger / Messenger overview** и datasource создаются автоматически.

В Compose метрики приложения доступны только в Docker-сети на app:9090. Порты Prometheus/Grafana опубликованы только на 127.0.0.1. Без профиля monitoring запускаются PostgreSQL, Redis, migrate и app. Redis не публикует порт на хост.

В volumes хранятся PostgreSQL, данные Prometheus и Grafana. Изменение начального пароля Grafana в .env не меняет пароль уже созданного администратора в существующем volume.

## Метрики и probes

| Адрес | Назначение |
| --- | --- |
| API /live | Процесс обслуживает HTTP; не зависит от БД/Redis |
| API /ready | PostgreSQL и настроенный Redis доступны; общий deadline 2 секунды |
| API /health | Совместимый alias readiness |
| METRICS_ADDR /metrics | Отдельный HTTP listener, по умолчанию 127.0.0.1:9090 |

Readiness исключает нездоровую реплику из Service, liveness проверяет сам процесс. Это соответствует [разделению Kubernetes probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

| Метрика | Что измеряет |
| --- | --- |
| messenger_http_requests_total | Завершённые HTTP-запросы: method, шаблон route, status |
| messenger_http_duration_seconds | Histogram длительности HTTP; длительность WebSocket-сессий исключена |
| messenger_websocket_connections | Активные соединения на каждой реплике |
| messenger_websocket_slow_clients_total | Отключения переполненных клиентских очередей |
| messenger_redis_publish_errors_total | Ошибки публикации событий между репликами |
| go_*, process_* | Runtime Go и процесс |

Идентификаторы пользователей/чатов, токены, содержимое сообщений и сырые URL в labels не записываются. /metrics не добавлен на публичный API-порт. Его listener не имеет аутентификации и предназначен для внутренней сети.

Дашборд показывает доступные реплики, RPS, 5xx, p95, WebSocket-соединения, ошибки Redis, goroutines, память и HTTP 429. Счётчики HTTP не считают отдельные сообщения внутри открытого WebSocket; ответ 101 учитывается после завершения handler.

В alerts.yml есть недоступность всех реплик, поток 5xx и ошибки публикации. Алерты видны в Prometheus; отправка уведомлений наружу не настроена (для неё нужен Alertmanager). Дашборд provisioning описан в [документации Grafana](https://grafana.com/docs/grafana/latest/administration/provisioning/).

## Kubernetes: первый запуск в kind

Нужны Docker, kubectl и [kind](https://kind.sigs.k8s.io/docs/user/quick-start/). Команды ниже выполняются из корня проекта; для production используй свой registry и уникальные image tags.

### 1. Кластер и образы

```bash
kind create cluster --name messenger
docker build --provenance=false --target app -t messenger-app:local .
docker build --provenance=false --target migrate -t messenger-migrate:local .
kind load docker-image messenger-app:local messenger-migrate:local --name messenger
kubectl apply -f deploy/k8s/infra/namespace.yaml
```

Для локальной загрузки в kind отключены дополнительные provenance-аттестации образа: это обходит ошибки импорта multi-platform manifest в некоторых версиях Docker Desktop/containerd. Для публикации в registry этот флаг не требуется.

### 2. Секреты

Создай локальные файлы deploy/messenger.secrets.env и deploy/grafana.secrets.env (они исключены из Git и Docker build context).

messenger.secrets.env:

```dotenv
DB_PASSWORD=replace-with-a-long-random-alphanumeric-password
JWT_SECRET=replace-with-a-long-random-secret
```

grafana.secrets.env:

```dotenv
GRAFANA_ADMIN_PASSWORD=replace-with-a-long-random-password
```

Используй собственные значения, одинаковый JWT_SECRET для всех реплик. Для DB_PASSWORD здесь рекомендован буквенно-цифровой пароль: существующий DSN builder проекта пока не экранирует специальные символы URL.

```bash
kubectl -n messenger create secret generic messenger-secrets --from-env-file=deploy/messenger.secrets.env
kubectl -n messenger create secret generic grafana-secrets --from-env-file=deploy/grafana.secrets.env
```

### 3. База, Redis и миграции

```bash
kubectl apply -k deploy/k8s/infra
kubectl -n messenger rollout status statefulset/postgres --timeout=180s
kubectl -n messenger rollout status deployment/redis --timeout=180s
kubectl apply -f deploy/k8s/migration-job.yaml
kubectl -n messenger wait --for=condition=complete job/messenger-migrate --timeout=180s
kubectl -n messenger logs job/messenger-migrate
```

Продолжай только после успешного Job. При ошибке сначала проверь его logs. Миграции не запускаются в каждой реплике; Job намеренно исключён из общего kustomization.

### 4. Две реплики и мониторинг

```bash
kubectl apply -k deploy
kubectl -n messenger rollout status deployment/messenger --timeout=180s
kubectl -n messenger rollout status deployment/prometheus --timeout=180s
kubectl -n messenger rollout status deployment/grafana --timeout=180s
kubectl -n messenger get pods
```

Prometheus через Kubernetes pod discovery собирает **каждую реплику**, а не случайную реплику за Service. Его ServiceAccount имеет read-only доступ к pods только в namespace messenger. ConfigMapGenerator обновляет имена конфигураций при изменении, вызывая rollout мониторинга.

Открой три отдельных терминала:

```bash
kubectl -n messenger port-forward svc/messenger 8080:8080
kubectl -n messenger port-forward svc/prometheus 9091:9090
kubectl -n messenger port-forward svc/grafana 3000:3000
```

Адреса те же, что для Compose. kubectl port-forward на Service выбирает один Pod и нужен для локальной демонстрации. Для реального внешнего трафика добавь Ingress/LoadBalancer с TLS и поддержкой WebSocket.

### Обновление

Собери и загрузи новые образы. В registry используй новый tag и обнови image в app.yaml и migration-job.yaml. Для local tags после загрузки нужен rollout restart.

Перед миграцией новой схемы удали **завершённый** старый Job и повтори шаг 3 с apply Job/wait. Не запускай несколько migration Jobs одновременно:

```bash
kubectl -n messenger delete job messenger-migrate
kubectl apply -f deploy/k8s/migration-job.yaml
kubectl -n messenger wait --for=condition=complete job/messenger-migrate --timeout=180s
kubectl apply -k deploy
kubectl -n messenger rollout restart deployment/messenger
kubectl -n messenger rollout status deployment/messenger
```

Схема БД при rolling update должна оставаться совместимой с работающей версией.

### Границы демо

- PostgreSQL — один StatefulSet с PVC 2 GiB; Redis — одна реплика. Это демонстрация масштабирования приложения, а не HA базы/Redis.
- Prometheus и Grafana в Kubernetes используют emptyDir: история метрик и изменения Grafana теряются при пересоздании Pod. Provisioned dashboard восстанавливается из Git. Для длительной эксплуатации нужны PVC, backup и подходящие retention/ресурсы.
- Redis доступен внутри кластера без пароля/TLS; манифесты предназначены для доверенного локального кластера. Для общего окружения потребуются сетевые ограничения и управление секретами.
- Изменение messenger-config или Secret не перезапускает приложение автоматически: после изменения делай rollout restart.
- Удаление namespace или kind-кластера уничтожит данные демо. Для очистки только этого стенда: kind delete cluster --name messenger.

## Проверка

При добавлении инфраструктуры проверены: go test/go vet, race detector с настоящими PostgreSQL/Redis, smoke test двух Docker-процессов и двух Pod во временном kind-кластере, успешный migration Job, два healthy scrape target, загрузка 10 панелей Grafana и всех PromQL-запросов, datasource health и три правила алертов. Сценарий недоступного Redis проверяется изолированными тестами (readiness, отказ отправки, сохранение локальной доставки).

Обычные тесты:

```bash
go test ./...
go vet ./...
```

Интеграционные тесты используют TEST_DATABASE_URL и TEST_REDIS_URL; без них соответствующие проверки пропускаются. CI поднимает PostgreSQL/Redis, запускает тесты с race detector и smoke test двух процессов.

Для проверки **двух конкретных Kubernetes Pod**, выведи их имена:

```bash
kubectl -n messenger get pods -l app=messenger
kubectl -n messenger port-forward pod/<first-pod> 8081:8080
kubectl -n messenger port-forward pod/<second-pod> 8082:8080
```

Два port-forward работают в отдельных терминалах. Затем Git Bash/Linux/macOS:

```bash
TEST_APP_A_URL=http://localhost:8081 TEST_APP_B_URL=http://localhost:8082 go test -race ./tests/infrastructure -v -count=1
```

PowerShell (если локально не установлен C compiler, запускай без -race; CI выполняет race test в Linux):

```powershell
$env:TEST_APP_A_URL="http://localhost:8081"
$env:TEST_APP_B_URL="http://localhost:8082"
go test ./tests/infrastructure -v -count=1
```

Тест создаёт пользователей, чат и сообщения: запускай на тестовой БД. Проверяет доставку между репликами, изменения/реакции, приватность mute, сохранение состояния и общий лимит 10/5s.

Проверка конфигураций без запуска приложения:

```bash
docker compose --profile monitoring config --quiet
kubectl kustomize deploy
docker run --rm --entrypoint promtool -v "${PWD}/deploy/monitoring:/etc/prometheus:ro" prom/prometheus:v3.5.0 check config /etc/prometheus/prometheus.yml
```

## Две последовательные ветки / PR

feature/redis-k8s-observability создана от локальной feature/reactions-and-mute на коммите 151c042. Поэтому новая ветка содержит предыдущую фичу как основание.

Сначала пуш и PR реакций/mute:

```bash
git push -u origin feature/reactions-and-mute
```

После слияния первого PR в main, особенно при squash merge, перенеси **только инфраструктурные коммиты** на обновлённый main:

```bash
git fetch origin
git rebase --onto origin/main 151c042 feature/redis-k8s-observability
git push -u origin feature/redis-k8s-observability
```

Затем создай второй PR в main. Выполняй rebase с чистой рабочей директорией; при конфликтах разреши их и выполни git rebase --continue. Эти команды приведены для дальнейшей работы; автоматически push/rebase на удалённый main не выполнялся.
