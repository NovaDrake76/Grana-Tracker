# Como contribuir

Obrigado por contribuir com o Grana Tracker. Este documento descreve o fluxo
mínimo esperado para manter o repositório consistente e revisável.

## Branching

Trabalhamos em modelo **trunk-based**:

- `main` é a branch protegida e sempre deployável.
- Feature branches saem de `main` e são abertas com prefixo:
  - `feat/<scope>` para novas funcionalidades
  - `fix/<scope>` para correções
  - `hotfix/<scope>` para correções urgentes em produção
- Integração é feita exclusivamente via Pull Request — nunca push direto na `main`.

## Commits

Seguimos [Conventional Commits](https://www.conventionalcommits.org/). Tipos em uso
neste repositório:

- `feat:` nova funcionalidade
- `fix:` correção de bug
- `test:` adição ou ajuste de testes
- `docs:` documentação
- `chore:` tarefas de manutenção
- `ci:` mudanças em pipeline / GitHub Actions

Confirme o estilo vigente com `git log --oneline` antes de abrir o PR.

## Pull Requests

- Mínimo de **1 reviewer** aprovando.
- **CI verde obrigatório** (job `backend build + tests` precisa passar).
- Use o template em `.github/PULL_REQUEST_TEMPLATE.md` — preencha todas as seções.
- Reviewers padrão são definidos em `.github/CODEOWNERS`.

## Testes locais

```bash
cd backend
go test ./...
```

Testes de integração precisam da variável `TEST_DATABASE_URL` apontando para
um Postgres acessível (use o `docker-compose.yml` na raiz para subir um local).

## Estilo de código

- `gofmt` obrigatório em todo arquivo Go (`gofmt -w .`).
- `go vet ./...` deve passar limpo antes do PR.
- Ao mexer em `backend/db/queries/`, rode `sqlc generate` e commit os arquivos
  gerados junto com a mudança.
