# Pixel Basketball 🏀

[Русский](#русский) | [English](#english)

---

## Русский

**Pixel Basketball** — браузерная мультиплеерная пиксельная баскетбольная игра 1 на 1 с видом сбоку.

### Особенности

- Мультиплеер в реальном времени через WebSocket
- Серверная авторитативная модель (60 тиков/сек)
- Пиксельная графика на Canvas
- Система никнеймов с сохранением в localStorage
- Мобильное управление (виртуальный джойстик + кнопка броска)
- Защита от дубликатов — нельзя играть с самим собой в одном браузере
- Запуск в Docker одной командой

### Технологии

| Компонент | Стек |
|-----------|------|
| Сервер | Go 1.25, WebSocket |
| Клиент | TypeScript, HTML5 Canvas |
| Сборка клиента | esbuild |
| Деплой | Docker, multi-stage build |

### Управление

**Клавиатура:**
| Действие | Игрок 1 | Игрок 2 |
|----------|---------|---------|
| Движение | A / D | ← / → |
| Прыжок | W | ↑ |
| Бросок | S | ↓ |

**Мобильные устройства:**
- Левая часть экрана — виртуальный джойстик (движение + прыжок)
- Правая часть экрана — кнопка броска

### Быстрый старт

**Docker (рекомендуется):**

```bash
docker compose up --build
```

Игра доступна по адресу: `http://localhost:8080`

**Для локальной разработки:**

```bash
# Установка зависимостей клиента
cd client && npm ci

# Сборка клиента + запуск сервера
cd .. && make dev
```

### Структура проекта

```
├── client/                 # Фронтенд (TypeScript + Canvas)
│   ├── src/
│   │   ├── game/           # Игровая логика (game.ts, touch.ts)
│   │   ├── network/        # WebSocket клиент (socket.ts, protocol.ts)
│   │   └── render/         # Рендеринг (renderer.ts)
│   ├── index.html          # Главная страница + оверлей никнейма
│   └── esbuild.config.mjs  # Конфигурация сборки
├── server/                 # Бэкенд (Go)
│   ├── cmd/server/         # Точка входа
│   └── internal/
│       ├── game/           # Игровая физика и логика (room.go)
│       └── ws/             # WebSocket хаб, подключения, сообщения
├── Dockerfile              # Multi-stage сборка
├── docker-compose.yml
└── Makefile
```

### Игра по сети (LAN)

Чтобы играть с другом по локальной сети:

1. Запустите сервер на одном компьютере
2. Узнайте IP-адрес хоста: `ifconfig | grep inet`
3. Второй игрок открывает: `http://<IP хоста>:8080`

---

## English

**Pixel Basketball** — a browser-based multiplayer pixel-art 1v1 basketball game with a side-view perspective.

### Features

- Real-time multiplayer via WebSocket
- Server-authoritative model (60 ticks/sec)
- Pixel-art graphics rendered on Canvas
- Nickname system with localStorage persistence
- Mobile controls (virtual joystick + shoot button)
- Duplicate protection — can't play against yourself from the same browser
- One-command Docker deployment

### Tech Stack

| Component | Stack |
|-----------|-------|
| Server | Go 1.25, WebSocket |
| Client | TypeScript, HTML5 Canvas |
| Bundler | esbuild |
| Deploy | Docker, multi-stage build |

### Controls

**Keyboard:**
| Action | Player 1 | Player 2 |
|--------|----------|----------|
| Move | A / D | ← / → |
| Jump | W | ↑ |
| Shoot | S | ↓ |

**Mobile:**
- Left side — virtual joystick (move + jump)
- Right side — shoot button

### Quick Start

**Docker (recommended):**

```bash
docker compose up --build
```

Game available at: `http://localhost:8080`

**Local development:**

```bash
# Install client dependencies
cd client && npm ci

# Build client + start server
cd .. && make dev
```

### Project Structure

```
├── client/                 # Frontend (TypeScript + Canvas)
│   ├── src/
│   │   ├── game/           # Game logic (game.ts, touch.ts)
│   │   ├── network/        # WebSocket client (socket.ts, protocol.ts)
│   │   └── render/         # Rendering (renderer.ts)
│   ├── index.html          # Main page + nickname overlay
│   └── esbuild.config.mjs  # Build config
├── server/                 # Backend (Go)
│   ├── cmd/server/         # Entry point
│   └── internal/
│       ├── game/           # Game physics & logic (room.go)
│       └── ws/             # WebSocket hub, connections, messages
├── Dockerfile              # Multi-stage build
├── docker-compose.yml
└── Makefile
```

### LAN Play

To play with a friend on a local network:

1. Start the server on one machine
2. Find the host IP: `ifconfig | grep inet`
3. The other player opens: `http://<host-IP>:8080`

---

*Built with Go, TypeScript, and Canvas. Made with Claude Code.*
