# AGENTS.md

# AgentMD

Этот файл является обязательной инструкцией для AI/code-agent, который будет проектировать, писать, рефакторить или тестировать проект Remote Admin MVP.

## Главная идея проекта

Remote Admin MVP — это не malware, не backdoor и не скрытый инструмент доступа. Это легальная RMM-like платформа для авторизованного удалённого администрирования устройств, где:

```text
AI-agent работает только на ПК администратора.
Клиентский ПК только исполняет явно отправленные jobs.
Все действия проходят через server API, audit log и policy layer.
```

Любая реализация должна предполагать наличие согласия владельца устройства, явную установку client-agent и возможность отзыва доступа.

## Архитектурные принципы

Проект должен строиться как набор независимых, строго типизированных компонентов:

```text
apps/server       Node.js + TypeScript
apps/admin-cli    Node.js + TypeScript
apps/admin-web    Next.js + TypeScript, позже
agent             Go
packages/protocol shared protocol contracts
packages/sdk-js   TypeScript SDK, позже
infra             deployment/dev infrastructure
```

Запрещено смешивать server, CLI, protocol и agent-логику в один монолит.

## Обязательные требования к качеству

Каждый production-компонент должен быть:

```text
- строго типизированным;
- модульным;
- покрытым unit/integration tests;
- наблюдаемым через structured logs;
- расширяемым без переписывания ядра;
- безопасным по умолчанию;
- документированным на уровне публичных interfaces;
- пригодным для локальной отладки.
```

TypeScript должен использовать strict mode. Запрещены `any`, неявные nullable-значения и невалидированные входные данные. Исключения допускаются только с коротким комментарием причины и отдельным issue/TODO.

Go-код должен проходить `go test ./...`, `go vet ./...` и быть разбит на небольшие packages по ответственности.

## Стиль проектирования для TypeScript-компонентов

Для `server`, `admin-cli`, `admin-web`, `sdk-js` использовать модульную архитектуру по feature/domain slices.

Рекомендуемая структура:

```text
src/
  app/              # composition root, DI, server bootstrap
  config/           # env/config parsing and validation
  shared/           # cross-cutting utils, logger, errors
  modules/
    devices/
      domain/
      application/
      infrastructure/
      presentation/
      tests/
    jobs/
    sessions/
    auth/
    audit/
```

Разделение слоёв:

```text
domain          чистые модели и бизнес-правила
application     use cases/services
infrastructure  database, transport, external integrations
presentation    HTTP/WebSocket/CLI handlers
```

Запрещено обращаться к БД напрямую из presentation layer. Запрещено помещать бизнес-логику в HTTP/WebSocket handlers.

## Стиль проектирования для Go agent

Go agent должен быть построен вокруг чистого core и platform adapters.

Рекомендуемая структура:

```text
agent/
  cmd/remote-agent/main.go
  internal/core/
  internal/transport/
  internal/executor/
  internal/platform/
    windows/
    linux/
  internal/config/
  internal/logging/
  internal/security/
  packaging/
    windows/
    linux/
```

Core не должен знать детали Windows или Linux. Все OS-specific операции должны быть спрятаны за interfaces.

Пример границ:

```text
core.JobRunner        orchestrates job lifecycle
executor.ShellRunner  runs commands with timeout/cancellation
platform.ServiceAPI   service status/restart abstraction
platform.LogsAPI      eventlog/journald abstraction
platform.RebootAPI    reboot abstraction
```

## Правила безопасности

Обязательно:

```text
- TLS для всех сетевых соединений;
- одноразовый enrollment token;
- device identity после enrollment;
- хранение private material в OS-secure storage, где возможно;
- audit log для всех jobs;
- timeout для каждой команды;
- ограничение размера stdout/stderr;
- отдельный approval flow для destructive actions;
- revoke для устройства;
- запрет silent/stealth поведения;
- явные имена service/process/config directories.
```

Запрещено реализовывать:

```text
- обход антивирусов/EDR;
- скрытую установку;
- persistence без явного service install;
- credential dumping;
- keylogging;
- screen capture без отдельного согласованного feature scope;
- отключение средств защиты;
- функции сокрытия процесса, файлов или сетевого трафика.
```

## Правила для AI-agent workflow

AI-agent должен использовать только публичный интерфейс проекта:

```text
remote-admin CLI
server API через официальный SDK
```

AI-agent не должен:

```text
- подключаться к клиентскому ПК напрямую;
- хранить reasoning/context на клиентском ПК;
- отправлять на клиентский ПК большие prompt-фрагменты;
- выполнять destructive actions без подтверждения администратора;
- обходить policy layer.
```

Правильный порядок диагностики:

```text
1. device info
2. session start
3. read-only диагностика
4. анализ результата локально
5. предложение изменений
6. approval для опасных действий
7. выполнение изменения
8. verify
9. session report
```

## Команды CLI должны быть machine-readable

Каждая команда CLI, используемая AI-agent, должна поддерживать:

```text
--json
--session <session-id>
--timeout <seconds>
--save
stable exit codes
```

Ошибки должны иметь предсказуемый формат:

```json
{
	"ok": false,
	"error": {
		"code": "DEVICE_OFFLINE",
		"message": "Device is offline",
		"retryable": true
	}
}
```

## Тестирование

Минимальные требования:

```text
- unit tests для domain/application logic;
- integration tests для API + database;
- contract tests для protocol;
- CLI tests для machine-readable output;
- Go tests для executor, reconnect, job lifecycle;
- platform adapter tests через mocks/fakes;
- e2e smoke сценарий: enroll → connect → shell.exec → result.
```

Нельзя считать feature готовой без тестов и обновления документации.

## Наблюдаемость

Каждый компонент должен писать structured logs. Нельзя логировать secrets, tokens, private keys, full credentials или персональные данные без маскирования.

Каждый job должен иметь correlation identifiers:

```text
tenant_id/org_id
admin_id
session_id
job_id
device_id
request_id
```

## Работа с протоколом

Любые изменения protocol contracts должны быть версионированы. Нельзя ломать совместимость agent/server без миграционного плана.

Рекомендуемый подход:

```text
packages/protocol/schemas
packages/protocol/generated
protocol_version в heartbeat
capabilities в device info
```

## Definition of Done

Feature считается готовой, если выполнено:

```text
- есть строгие типы/контракты;
- есть тесты;
- есть structured logs;
- есть обработка ошибок;
- есть документация;
- нет secrets в логах;
- нет неявных bypass путей;
- код разбит по слоям;
- команда запуска/проверки описана;
- локальная отладка воспроизводима.
```

## Приоритеты MVP

Первым делом реализовывать:

```text
1. protocol contracts;
2. server devices/jobs/sessions/audit;
3. Go agent enroll/connect/heartbeat/shell.exec;
4. admin-cli devices/session/shell;
5. local session artifacts;
6. tests and docs.
```

Не начинать с remote desktop, сложного UI, stealth features или интерактивного shell. MVP должен быть job-based.

Before making changes, read:

```text
AgentMD.md
docs/00-index.md
docs/02-system-architecture.md
docs/03-mvp-scope.md
docs/07-protocol.md
docs/08-security-and-authorization.md
```

Follow `AgentMD.md` as the source of truth for architecture, typing, testing, security and implementation style.
