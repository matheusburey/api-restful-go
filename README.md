# API RESTFUL GO-CHI

API RESTful em Go com Chi, PostgreSQL e WebSockets: cadastro/autenticação de usuários e um sistema de leilões em tempo real (criação de produtos, lances via WebSocket).

### 🔧 Instalação
Clone o repositório e instale as dependências:
```sh
  git clone git@github.com:matheusburey/api_restful.git
  cd api_restful
  go mod download
```

Copie o arquivo de variáveis de ambiente e ajuste se necessário:
```sh
  cp .env.example .env
```

Se não tiver um banco de dados, suba um via Docker Compose:
```sh
  docker-compose up -d
```

### 🚀 Execução
Rodando as migrações:
```sh
  make migrate
```

Para iniciar o servidor:
```sh
  make run
```

Para desenvolvimento com hot-reload (requer [air](https://github.com/air-verse/air) instalado):
```sh
  make dev
```

Veja todos os comandos disponíveis com `make help` (migrate, migrate-generate, sqlc, run, dev, test, tidy, clean).

### 🧪 Testes
```sh
  make test
```
equivalente a `go test ./...`. Para ver a cobertura:
```sh
  go test ./... -cover
```
Os testes de `internal/services` e `internal/api` não dependem de um banco de dados real — usam fakes de `internal/store/pgstore/pgstoretest` que implementam apenas o subconjunto de `pgstore.Queries` que cada camada consome.

### 🛠️ Construído com

* [Go](https://go.dev/) - Linguagem de programação principal;
* [Go-chi](https://go-chi.io/#/README) - Framework web leve e idiomático para Go;
* [pgx](https://github.com/jackc/pgx) - Driver PostgreSQL avançado para Go;
* [sqlc](https://sqlc.dev/) - Gerador de código que cria queries tipadas a partir de SQL puro;
* [tern](https://github.com/jackc/tern) - Ferramenta de versionamento e gerenciamento de migrações de banco de dados;
* [scs](https://github.com/alexedwards/scs) - Gerenciamento de sessões HTTP (autenticação baseada em cookie de sessão);
* [gorilla/websocket](https://github.com/gorilla/websocket) - Implementação de WebSocket para os lances em tempo real dos leilões;
* [godotenv](https://github.com/joho/godotenv) - Carregamento de variáveis de ambiente a partir de `.env`;
* [Docker/Docker Compose](https://www.docker.com/) - Gerenciador de contêineres;

### 🔐 Autenticação

A autenticação é baseada em sessão via cookie. `POST /api/v1/users/login` renova o token de sessão e devolve um cookie de sessão que deve ser enviado (automaticamente pelo navegador, ou manualmente via `-b`/`-c` no `curl`) nas rotas protegidas abaixo (marcadas com 🔒).

### EndPoint

URL_BASE = `http://localhost:3333`

Todas as respostas seguem o formato `{ "message"?, "error"?, "data"? }`.

### 🔹 Cadastrar usuário
- **POST** `/api/v1/users/signup`
  **Body (JSON):**
  ```json
  {
    "name": "user 3",
    "email": "usuario@email.com",
    "password": "Abc123@@",
    "bio": "uma bio com pelo menos 10 caracteres"
  }
  ```
  **Response success:** `201 Created` — `{ "message": "success" }`

  **Response error:** `422 Unprocessable Entity`
  ```json
  { "error": "email already registered" }
  ```

  **Response error:** `400 Bad Request`
  ```json
  { "error": "bad request", "message": { "password": "min length is 8" } }
  ```

### 🔹 Login
- **POST** `/api/v1/users/login`
  **Body (JSON):**
  ```json
  { "email": "usuario@email.com", "password": "Abc123@@" }
  ```
  **Response success:** `200 OK` — `{ "message": "success" }` (define o cookie de sessão)

  **Response error:** `401 Unauthorized`
  ```json
  { "error": "invalid credentials" }
  ```

### 🔹 Logout 🔒
- **POST** `/api/v1/users/logout`
  **Response:** `200 OK` — `{ "message": "success" }`

### 🔹 Atualizar usuário autenticado 🔒
- **PUT** `/api/v1/users/`
  Atualiza os dados do usuário da sessão atual (o `password` é opcional; se omitido, a senha atual é mantida).
  **Body (JSON):**
  ```json
  {
    "name": "user 3",
    "email": "usuario@email.com",
    "bio": "bio atualizada com pelo menos 10 caracteres"
  }
  ```
  **Response success:** `200 OK`
  ```json
  {
    "data": {
      "id": "80b30bd9-5645-4a24-95b4-063b0cee15f3",
      "name": "user 3",
      "email": "usuario@email.com",
      "bio": "bio atualizada com pelo menos 10 caracteres",
      "created_at": "2025-04-24T01:08:07.05422",
      "updated_at": "2025-04-24T01:08:07.05422"
    }
  }
  ```

  **Response error:** `401 Unauthorized` — sem sessão válida.

### 🔹 Deletar usuário autenticado 🔒
- **DELETE** `/api/v1/users/`
  Deleta o usuário da sessão atual.
  **Response:** `204 No Content`

### 🔹 Criar produto (inicia um leilão) 🔒
- **POST** `/api/v1/products/`
  `auction_end` precisa ser pelo menos 2 horas no futuro.
  **Body (JSON):**
  ```json
  {
    "name": "Relógio Vintage",
    "description": "um relógio muito bonito e antigo",
    "base_price_cents": 5000,
    "auction_end": "2026-08-01T20:00:00Z"
  }
  ```
  **Response success:** `201 Created`
  ```json
  {
    "message": "Auction has started with success",
    "data": { "product_id": "80b30bd9-5645-4a24-95b4-063b0cee15f3" }
  }
  ```

### 🔹 Acompanhar leilão em tempo real (WebSocket) 🔒
- **GET** `/api/v1/products/ws/subscribe/{product_id}`
  Faz upgrade da conexão para WebSocket e conecta o usuário à sala do leilão do produto, para enviar lances e receber atualizações em tempo real enquanto o leilão estiver ativo.

  **Response error:** `404 Not Found` — produto inexistente ou leilão já encerrado.
